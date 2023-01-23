package util

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"oras.land/oras-go/pkg/registry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/kind/pkg/cluster"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"

	pkgUtil "github.com/Azure/eraser/pkg/utils"
)

const (
	providerResourceChartDir  = "manifest_staging/charts"
	providerResourceDeployDir = "manifest_staging/deploy"

	KindClusterName  = "eraser-e2e-test"
	ProviderResource = "eraser.yaml"

	Alpine        = "alpine"
	Nginx         = "nginx"
	NginxLatest   = "docker.io/library/nginx:latest"
	NginxAliasOne = "docker.io/library/nginx:one"
	NginxAliasTwo = "docker.io/library/nginx:two"
	Redis         = "redis"
	Caddy         = "caddy"

	ImageCollectorShared = "imagecollector-shared"
	Prune                = "imagelist"
	ImagePullSecret      = "testsecret"
	FilterNodeName       = "eraser-e2e-test-worker"
	FilterNodeSelector   = "kubernetes.io/hostname=eraser-e2e-test-worker"
	FilterLabelKey       = "eraser.sh/cleanup.filter"
	FilterLabelValue     = "true"
)

const (
	CollectorEnable    = HelmPath("runtimeConfig.components.collector.enabled")
	CollectorImageRepo = HelmPath("runtimeConfig.components.collector.image.repo")
	CollectorImageTag  = HelmPath("runtimeConfig.components.collector.image.tag")

	ScannerEnable    = HelmPath("runtimeConfig.components.scanner.enabled")
	ScannerImageRepo = HelmPath("runtimeConfig.components.scanner.image.repo")
	ScannerImageTag  = HelmPath("runtimeConfig.components.scanner.image.tag")

	EraserImageRepo = HelmPath("runtimeConfig.components.eraser.image.repo")
	EraserImageTag  = HelmPath("runtimeConfig.components.eraser.image.tag")

	ManagerImageRepo = HelmPath("deploy.image.repo")
	ManagerImageTag  = HelmPath("deploy.image.tag")

	ImagePullSecrets = HelmPath("runtimeConfig.manager.pullSecrets")
	OTLPEndpoint     = HelmPath("runtimeConfig.manager.otlpEndpoint")

	CleanupOnSuccessDelay = HelmPath("runtimeConfig.manager.imageJob.cleanup.delayOnSuccess")
	FilterNodesType       = HelmPath("runtimeConfig.manager.nodeFilter.type")
	ScheduleImmediate     = HelmPath("runtimeConfig.manager.scheduling.beginImmediately")
)

var (
	Testenv            env.Environment
	Image              = os.Getenv("IMAGE")
	ManagerImage       = os.Getenv("MANAGER_IMAGE")
	CollectorImage     = os.Getenv("COLLECTOR_IMAGE")
	ScannerImage       = os.Getenv("SCANNER_IMAGE")
	VulnerableImage    = os.Getenv("VULNERABLE_IMAGE")
	NonVulnerableImage = os.Getenv("NON_VULNERABLE_IMAGE")
	NodeVersion        = os.Getenv("NODE_VERSION")
	TestNamespace      = envconf.RandomName("test-ns", 16)
	EraserNamespace    = pkgUtil.GetNamespace()
	TestLogDir         = os.Getenv("TEST_LOGDIR")

	ParsedImages        *Images
	Timeout             = time.Minute * 5
	ImagePullSecretJSON = fmt.Sprintf(`["%s"]`, ImagePullSecret)

	ManagerAdditionalArgs = HelmSet{
		key:  "controllerManager.additionalArgs",
		args: []string{"--delete-scan-failed-images=false"},
	}
)

type (
	RepoTag struct {
		Repo string
		Tag  string
	}

	Images struct {
		CollectorImage RepoTag
		EraserImage    RepoTag
		ManagerImage   RepoTag
		ScannerImage   RepoTag
	}

	HelmPath string

	HelmSet struct {
		key  string
		args []string
	}
)

func (hp HelmPath) Set(val string) string {
	return fmt.Sprintf("%s=%s", hp, val)
}

func (hs *HelmSet) Set(val ...string) *HelmSet {
	hs.args = append(hs.args, val...)
	return hs
}

func (hs *HelmSet) String() string {
	return fmt.Sprintf("%s={%s}", hs.key, strings.Join(hs.args, ","))
}

func init() {
	var err error
	ParsedImages, err = parsedImages(Image, ManagerImage, CollectorImage, ScannerImage)
	if err != nil {
		klog.Error(err)
		panic(err)
	}
}

func toRepoTag(ref registry.Reference) RepoTag {
	var repoTag RepoTag

	repoTag.Repo = fmt.Sprintf("%s/%s", ref.Registry, ref.Repository)
	if repoTag.Repo == "/" {
		repoTag.Repo = ""
	}

	repoTag.Tag = ref.Reference
	return repoTag
}

func parsedImages(eraserImage, managerImage, collectorImage, scannerImage string) (*Images, error) {
	eraserRepoTag, err := parseRepoTag(eraserImage)
	if err != nil {
		return nil, err
	}

	collectorRepoTag, err := parseRepoTag(collectorImage)
	if err != nil {
		return nil, err
	}

	managerRepoTag, err := parseRepoTag(managerImage)
	if err != nil {
		return nil, err
	}

	scannerRepoTag, err := parseRepoTag(scannerImage)
	if err != nil {
		return nil, err
	}

	return &Images{
		CollectorImage: collectorRepoTag,
		EraserImage:    eraserRepoTag,
		ManagerImage:   managerRepoTag,
		ScannerImage:   scannerRepoTag,
	}, nil
}

func parseRepoTag(img string) (RepoTag, error) {
	if img == "" {
		return RepoTag{}, nil
	}

	ref, err := registry.ParseReference(img)
	if err == nil {
		return toRepoTag(ref), nil
	}

	// if true, this is an "unpublished" image, without a registry
	if parts := strings.Split(img, "/"); len(parts) == 1 {
		// the parser doesn't like unpublished images, so supply a dummy registry and pass it back to the parser
		var result registry.Reference
		result, err = registry.ParseReference(fmt.Sprintf("dummy.co/%s", img))
		if err == nil {
			return RepoTag{
				// the registry info is discarded since it was a dummy registry
				Repo: result.Repository,
				Tag:  result.Reference,
			}, nil
		}
	}

	return RepoTag{}, err
}

func IsNotFound(err error) bool {
	return err != nil && client.IgnoreNotFound(err) == nil
}

func NewDeployment(namespace, name string, replicas int32, labels map[string]string, containers ...corev1.Container) *appsv1.Deployment {
	if len(containers) == 0 {
		containers = []corev1.Container{
			{Image: "nginx", Name: "nginx"},
		}
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: labels,
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					Containers: containers,
				},
			},
		},
	}
}

func NewPod(namespace, image, name, nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: image,
				},
			},
		},
	}
}

// deploy eraser config.
func DeployEraserConfig(kubeConfig, namespace, resourcePath, fileName string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	exampleResourceAbsolutePath, err := filepath.Abs(filepath.Join(wd, resourcePath))
	if err != nil {
		return err
	}
	errApply := KubectlApply(kubeConfig, namespace, []string{"-f", filepath.Join(exampleResourceAbsolutePath, fileName)})
	if errApply != nil {
		return errApply
	}

	return nil
}

func NumPodsPresentForLabel(ctx context.Context, client klient.Client, num int, label string) func() (bool, error) {
	return func() (bool, error) {
		var pods corev1.PodList
		err := client.Resources().List(ctx, &pods, resources.WithLabelSelector(label))
		if err != nil {
			return false, err
		}

		return len(pods.Items) == num, nil
	}
}

func ContainerNotPresentOnNode(nodeName, containerName string) func() (bool, error) {
	return func() (bool, error) {
		output, err := ListNodeContainers(nodeName)
		if err != nil {
			return false, err
		}

		return !strings.Contains(output, containerName), nil
	}
}

func ImagejobNotInCluster(kubeconfigPath string) func() (bool, error) {
	return func() (bool, error) {
		output, err := KubectlGet(kubeconfigPath, "imagejob")
		if err != nil {
			return false, err
		}

		return strings.Contains(output, "No resources"), nil
	}
}

func GetImageJob(ctx context.Context, cfg *envconf.Config) (eraserv1alpha1.ImageJob, error) {
	c, err := cfg.NewClient()
	if err != nil {
		return eraserv1alpha1.ImageJob{}, err
	}

	var ls eraserv1alpha1.ImageJobList
	err = c.Resources().List(ctx, &ls)
	if err != nil {
		return eraserv1alpha1.ImageJob{}, err
	}

	if len(ls.Items) != 1 {
		return eraserv1alpha1.ImageJob{}, errors.New("only one imagejob should be present")
	}

	return ls.Items[0], nil
}

func ListNodeContainers(nodeName string) (string, error) {
	args := []string{
		"exec",
		nodeName,
		"ctr",
		"-n",
		"k8s.io",
		"containers",
		"list",
	}

	cmd := exec.Command("docker", args...)
	stdoutStderr, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdoutStderr))
	if err != nil {
		err = fmt.Errorf("%w: %s", err, output)
	}

	return output, err
}

func ListNodeImages(nodeName string) (string, error) {
	args := []string{
		"exec",
		nodeName,
		"ctr",
		"-n",
		"k8s.io",
		"images",
		"list",
	}

	cmd := exec.Command("docker", args...)
	stdoutStderr, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdoutStderr))
	if err != nil {
		err = fmt.Errorf("%w: %s", err, output)
	}

	return output, err
}

// This lists nodes in the cluster, filtering out the control-plane.
func GetClusterNodes(t *testing.T) []string {
	t.Helper()
	provider := cluster.NewProvider(cluster.ProviderWithDocker())

	nodeList, err := provider.ListNodes(KindClusterName)
	if err != nil {
		t.Fatal("Cannot list Kind node list", err)
	}
	var ourNodes []string
	for i := range nodeList {
		n := nodeList[i].String()
		if !strings.Contains(n, "control-plane") {
			ourNodes = append(ourNodes, n)
		}
	}

	return ourNodes
}

func CheckImagesExist(ctx context.Context, t *testing.T, nodes []string, images ...string) {
	t.Helper()

	for _, node := range nodes {
		nodeImages, err := ListNodeImages(node)
		if err != nil {
			t.Errorf("Cannot list images on node %s: %v", node, err)
			continue
		}

		for _, image := range images {
			if !strings.Contains(nodeImages, image) {
				t.Errorf("image %s missing on node %s", image, node)
			}
		}
	}
}

func CheckDeploymentCleanedUp(ctx context.Context, t *testing.T, client klient.Client) {
	t.Helper()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var pods corev1.PodList
			err := client.Resources().List(ctx, &pods, resources.WithLabelSelector("name=eraser"))
			if err != nil {
				t.Fatalf("error listing images: %s", err)
			}

			if len(pods.Items) > 0 {
				t.Errorf("imagejob got restarted when it shouldn't: %d eraser pods still present", len(pods.Items))
				t.FailNow()
			}
		}
		time.Sleep(time.Second * 2)
	}
}

func CheckImageRemoved(ctx context.Context, t *testing.T, nodes []string, images ...string) {
	t.Helper()

	cleaned := make(map[string]bool)
	for len(cleaned) < len(nodes) {
		select {
		case <-ctx.Done():
			t.Error("timeout waiting for images to be cleaned")
			return
		default:
		}
		for _, node := range nodes {
			done := cleaned[node]
			if done {
				continue
			}

			nodeImages, err := ListNodeImages(node)
			if err != nil {
				t.Error("Cannot list images", err)
			}

			var found int
			for _, img := range images {
				if !strings.Contains(nodeImages, img) {
					found++
				}
			}

			if found == len(images) {
				cleaned[node] = true
			}
		}
		time.Sleep(time.Second)
	}

	if len(cleaned) < len(nodes) {
		t.Error("not all nodes cleaned")
	}
}

func DockerPullImage(image string) (string, error) {
	args := []string{"pull", image}
	cmd := exec.Command("docker", args...)

	stdoutStderr, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdoutStderr))
	if err != nil {
		err = fmt.Errorf("%w: %s", err, output)
	}

	return output, err
}

func DockerTagImage(image, tag string) (string, error) {
	args := []string{"tag", image, tag}
	cmd := exec.Command("docker", args...)

	stdoutStderr, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdoutStderr))
	if err != nil {
		err = fmt.Errorf("%w: %s", err, output)
	}

	return output, err
}

func KindLoadImage(clusterName string, images ...string) (string, error) {
	args := []string{"load", "docker-image", "--name", clusterName}
	args = append(args, images...)
	cmd := exec.Command("kind", args...)

	stdoutStderr, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdoutStderr))
	if err != nil {
		err = fmt.Errorf("%w: %s", err, output)
	}

	return output, err
}

func DeleteImageListsAndJobs(kubeConfig string) error {
	if err := KubectlDelete(kubeConfig, "", []string{"imagejob", "--all"}); err != nil {
		return err
	}
	if err := KubectlDelete(kubeConfig, "", []string{"imagelist", "--all"}); err != nil {
		return err
	}
	return nil
}

func DeleteStringFromSlice(strings []string, s string) []string {
	idx := -1
	for i, cmp := range strings {
		if cmp == s {
			idx = i
			break
		}
	}

	if idx >= 0 {
		l := len(strings)
		strings[l-1], strings[idx] = strings[idx], strings[l-1]
		return strings[:l-1]
	}

	return strings
}

func DeployEraserHelm(namespace string, args ...string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		wd, err := os.Getwd()
		if err != nil {
			return ctx, err
		}

		providerResourceAbsolutePath, err := filepath.Abs(filepath.Join(wd, "../../../../", providerResourceChartDir, "eraser"))
		if err != nil {
			return ctx, err
		}

		emptyValuesPath, err := filepath.Abs(filepath.Join(wd, "../../test-data/helm-empty-values.yaml"))
		if err != nil {
			return ctx, err
		}

		// start deployment
		allArgs := []string{providerResourceAbsolutePath, "-f", emptyValuesPath}
		allArgs = append(allArgs, args...)
		if err := HelmInstall(cfg.KubeconfigFile(), namespace, allArgs); err != nil {
			return ctx, err
		}

		client, err := cfg.NewClient()
		if err != nil {
			klog.ErrorS(err, "Failed to create new Client")
			return ctx, err
		}

		// wait for the deployment to finish becoming available
		eraserManagerDep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "eraser-controller-manager", Namespace: namespace},
		}

		if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&eraserManagerDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
			wait.WithTimeout(Timeout)); err != nil {
			klog.ErrorS(err, "failed to deploy eraser manager")

			return ctx, err
		}

		return ctx, nil
	}
}

func DeployOtelCollector(namespace string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// set up otel collector
		wd, err := os.Getwd()
		if err != nil {
			return ctx, err
		}

		otelCollectorAbsolutePath, err := filepath.Abs(filepath.Join(wd, "../../", "test-data/otelcollector.yaml"))
		if err != nil {
			return ctx, err
		}

		// start otelcollector deployment
		otelargs := []string{"-f", otelCollectorAbsolutePath}
		if err := KubectlApply(cfg.KubeconfigFile(), namespace, otelargs); err != nil {
			return ctx, err
		}

		client, err := cfg.NewClient()
		if err != nil {
			klog.ErrorS(err, "Failed to create new Client")
			return ctx, err
		}

		// wait for the deployment to finish becoming available
		otelCollectorDep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "otel-collector", Namespace: namespace},
		}

		if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&otelCollectorDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
			wait.WithTimeout(Timeout)); err != nil {
			klog.ErrorS(err, "failed to deploy otelcollector")

			return ctx, err
		}

		return ctx, nil
	}
}

func GetManagerLogs(ctx context.Context, cfg *envconf.Config, t *testing.T) error {
	c, err := cfg.NewClient()
	if err != nil {
		return err
	}

	var pods corev1.PodList
	err = c.Resources().List(ctx, &pods, func(o *metav1.ListOptions) {
		o.LabelSelector = labels.SelectorFromSet(map[string]string{"control-plane": "controller-manager"}).String()
	})
	if err != nil {
		return err
	}

	if len(pods.Items) > 1 {
		return errors.New("only one manager pod should be present")
	} else if len(pods.Items) == 0 {
		return errors.New("no manager pod present")
	}

	manager := pods.Items[0]

	output, err := KubectlLogs(cfg.KubeconfigFile(), manager.Name, "", cfg.Namespace())
	if err != nil {
		return err
	}

	testName := strings.Split(t.Name(), "/")[0]

	// get log output file path
	path := filepath.Join(TestLogDir, testName)
	logFileName := filepath.Join(path, manager.Name+".txt")

	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}

	err = os.WriteFile(logFileName, []byte(output), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func GetPodLogs(ctx context.Context, cfg *envconf.Config, t *testing.T, imagelistTest bool) error {
	c, err := cfg.NewClient()
	if err != nil {
		return err
	}

	namespace := cfg.Namespace()

	labelSelectorSet := map[string]string{"name": "collector"}
	if imagelistTest {
		labelSelectorSet = map[string]string{"name": "eraser"}
	}

	var ls corev1.PodList
	err = wait.For(func() (bool, error) {
		err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
			o.LabelSelector = labels.SelectorFromSet(labelSelectorSet).String()
		})

		if err != nil {
			return false, err
		}

		return len(ls.Items) > 0, nil
	}, wait.WithTimeout(Timeout))

	if err != nil {
		t.Errorf("could not list pods: %v", err)
	}

	for idx := range ls.Items {
		var output string
		pod := ls.Items[idx]

		testName := strings.Split(t.Name(), "/")[0]

		// get log output file path
		path := filepath.Join(TestLogDir, testName)
		podLogFilename := filepath.Join(path, pod.Name+".txt")

		// wait for current pod to complete
		err = wait.For(conditions.New(c.Resources()).PodPhaseMatch(&pod, corev1.PodSucceeded), wait.WithTimeout(Timeout))
		if err != nil {
			t.Errorf("error waiting for pod completion %s %v", pod.Name, err)
		}

		output, err := KubectlLogs(cfg.KubeconfigFile(), pod.Name, "", namespace, "--all-containers=true")
		if err != nil {
			t.Errorf("error getting pod logs %s %v", pod.Name, err)
		}

		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}

		err = os.WriteFile(podLogFilename, []byte(output), 0o600)
		if err != nil {
			t.Errorf("error writing pod log file %s %s %v", pod.Name, podLogFilename, err)
		}
	}

	return nil
}

func MakeDeploy(env map[string]string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		args := []string{"deploy"}
		for k, v := range env {
			args = append(args, fmt.Sprintf("%s=%s", k, v))
		}

		cmd := exec.Command("make", args...)
		cmd.Dir = "../../../.."

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprint(os.Stderr, string(out))
			return ctx, err
		}

		klog.Info(string(out))

		return ctx, nil
	}
}

func DeployEraserManifest(namespace, fileName string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		providerResourceAbsolutePath := "../../../../" + providerResourceDeployDir

		if err := DeployEraserConfig(cfg.KubeconfigFile(), namespace, providerResourceAbsolutePath, fileName); err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}

func CreateExclusionList(namespace string, list string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		c, err := cfg.NewClient()
		if err != nil {
			return ctx, err
		}

		// create excluded configmap and add docker.io/library/alpine
		excluded := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "excluded",
				Namespace:    namespace,
				Labels:       map[string]string{"eraser.sh/exclude.list": "true"},
			},
			Data: map[string]string{"excluded.json": list},
		}
		if err := cfg.Client().Resources().Create(ctx, &excluded); err != nil {
			return ctx, err
		}

		cMap := corev1.ConfigMap{}
		err = wait.For(func() (bool, error) {
			err := c.Resources().Get(ctx, excluded.Name, namespace, &cMap)
			if IsNotFound(err) {
				return false, nil
			}

			if err != nil {
				return false, err
			}

			if cMap.ObjectMeta.Name == excluded.Name {
				return true, nil
			}

			return false, nil
		}, wait.WithTimeout(Timeout))
		if err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}
