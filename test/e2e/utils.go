//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kind/pkg/cluster"
)

func newDeployment(namespace, name string, replicas int32, labels map[string]string, containers ...corev1.Container) *appsv1.Deployment {
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

// deploy eraser config
func deployEraserConfig(kubeConfig, namespace, resourcePath, fileName string) error {
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

// delete eraser config
func deleteEraserConfig(kubeConfig, namespace, resourcePath, fileName string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	exampleResourceAbsolutePath, err := filepath.Abs(filepath.Join(wd, resourcePath))
	if err != nil {
		return err
	}
	errDelete := KubectlDelete(kubeConfig, namespace, []string{"-f", filepath.Join(exampleResourceAbsolutePath, fileName)})
	if errDelete != nil {
		return errDelete
	}

	return nil
}

func listNodeImages(nodeName string) (string, error) {
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
	return strings.TrimSpace(string(stdoutStderr)), err
}

// This lists nodes in the cluster, filtering out the control-plane
func getClusterNodes(t *testing.T) []string {
	t.Helper()
	provider := cluster.NewProvider(cluster.ProviderWithDocker())

	nodeList, err := provider.ListNodes(kindClusterName)
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

func checkImagesExist(ctx context.Context, t *testing.T, nodes []string, images ...string) {
	t.Helper()

	for _, node := range nodes {
		nodeImages, err := listNodeImages(node)
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

func checkImageRemoved(ctx context.Context, t *testing.T, nodes []string, images ...string) {
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

			nodeImages, err := listNodeImages(node)
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
