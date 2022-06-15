/*
Copyright 2021.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package imagejob

import (
	"context"
	"flag"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	eraserv1alpha1 "github.com/Azure/eraser/api/eraser.sh/v1alpha1"
	"github.com/Azure/eraser/controllers/util"
	"github.com/Azure/eraser/pkg/utils"
)

const (
	dockerPath     = "/run/dockershim.sock"
	containerdPath = "/run/containerd/containerd.sock"
	crioPath       = "/run/crio/crio.sock"
	docker         = "docker"
	containerd     = "containerd"
	crio           = "cri-o"
	namespace      = "eraser-system"
)

var log = logf.Log.WithName("controller").WithValues("process", "imagejob-controller")

var (
	successRatio       = flag.Float64("job-success-ratio", 1.0, "Ratio of successful/total runs to consider a job successful. 1.0 means all runs must succeed.")
	skipNodesSelectors = utils.MultiFlag([]string{"kubernetes.io/os=windows", "eraser.sh/cleanup.skip"})
)

func init() {
	flag.Var(&skipNodesSelectors, "skip-nodes-selector", "A kubernetes selector. If a node's labels are a match, the node will be skipped. If this flag is supplied multiple times, the selectors will be logically ORed together.")
}

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		Client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		successRatio: *successRatio,
	}
}

// ImageJobReconciler reconciles a ImageJob object.
type Reconciler struct {
	client.Client
	scheme *runtime.Scheme

	successRatio float64
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("imagejob-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch for changes to ImageJob
	err = c.Watch(&source.Kind{Type: &eraserv1alpha1.ImageJob{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if job, ok := e.ObjectNew.(*eraserv1alpha1.ImageJob); ok && util.IsCompletedOrFailed(job.Status.Phase) {
				return false // handled by Owning controller
			}

			return true
		},
		CreateFunc:  util.AlwaysOnCreate,
		GenericFunc: util.NeverOnGeneric,
		DeleteFunc:  util.NeverOnDelete,
	})
	if err != nil {
		return err
	}

	// Watch for changes to pods created by ImageJob (eraser pods)
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{OwnerType: &eraserv1alpha1.ImageJob{}, IsController: true})
	if err != nil {
		return err
	}

	return nil
}

func checkNodeFitness(pod *corev1.Pod, node *corev1.Node) bool {
	nodeInfo := framework.NewNodeInfo()
	nodeInfo.SetNode(node)

	insufficientResource := noderesources.Fits(pod, nodeInfo, feature.DefaultFeatureGate.Enabled(features.PodOverhead))

	if len(insufficientResource) != 0 {
		log.Error(fmt.Errorf("pod %v in namespace %v does not fit in node %v", pod.Name, pod.Namespace, node.Name), "insufficient resource")
		return false
	}

	return true
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;create;delete

//+kubebuilder:rbac:groups=eraser.sh,resources=imagecollectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagecollectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagecollectors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	imageJob := &eraserv1alpha1.ImageJob{}
	if err := r.Get(ctx, req.NamespacedName, imageJob); err != nil {
		imageJob.Status.Phase = eraserv1alpha1.PhaseFailed
		if err := r.updateJobStatus(ctx, imageJob); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	switch imageJob.Status.Phase {
	case "":
		if err := r.handleNewJob(ctx, imageJob); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconcile new: %w", err)
		}
	case eraserv1alpha1.PhaseRunning:
		if err := r.handleRunningJob(ctx, imageJob); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconcile running: %w", err)
		}
	case eraserv1alpha1.PhaseCompleted, eraserv1alpha1.PhaseFailed:
		break // this is handled by the Owning controller
	default:
		return ctrl.Result{}, fmt.Errorf("reconcile: unexpected imagejob phase: %s", imageJob.Status.Phase)
	}

	return ctrl.Result{}, nil
}

func podListOptions(j *eraserv1alpha1.ImageJob) client.ListOptions {
	return client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{"name": j.Spec.JobTemplate.Spec.Containers[0].Name}),
	}
}

func (r *Reconciler) handleRunningJob(ctx context.Context, imageJob *eraserv1alpha1.ImageJob) error {
	// get eraser pods
	podList := &corev1.PodList{}
	listOpts := podListOptions(imageJob)
	err := r.List(ctx, podList, &listOpts)
	if err != nil {
		return err
	}

	failed := 0
	success := 0
	skipped := imageJob.Status.Skipped

	if !podsComplete(podList.Items) {
		return nil
	}

	// if all pods are complete, job is complete
	// get status of pods
	for i := range podList.Items {
		if podList.Items[i].Status.Phase == corev1.PodSucceeded {
			success++
		} else {
			failed++
		}
	}

	imageJob.Status = eraserv1alpha1.ImageJobStatus{
		Desired:   imageJob.Status.Desired,
		Succeeded: success,
		Skipped:   skipped,
		Failed:    failed,
		Phase:     eraserv1alpha1.PhaseCompleted,
	}

	successAndSkipped := success + skipped
	if float64(successAndSkipped/imageJob.Status.Desired) < r.successRatio {
		log.Info("Marking job as failed", "success ratio", r.successRatio, "actual ratio", success/imageJob.Status.Desired)
		imageJob.Status.Phase = eraserv1alpha1.PhaseFailed
	}

	if err := r.updateJobStatus(ctx, imageJob); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) handleNewJob(ctx context.Context, imageJob *eraserv1alpha1.ImageJob) error {
	nodes := &corev1.NodeList{}
	err := r.List(ctx, nodes)
	if err != nil {
		return err
	}

	imageJob.Status = eraserv1alpha1.ImageJobStatus{
		Desired:   len(nodes.Items),
		Succeeded: 0,
		Skipped:   0, // placeholder, updated below
		Failed:    0,
		Phase:     eraserv1alpha1.PhaseRunning,
	}

	skipped := 0

	log := log.WithValues("job", imageJob.Name)

	env := []corev1.EnvVar{
		{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
	}

	nodeList, skipped, err := filterOutSkippedNodes(nodes, skipNodesSelectors)
	if err != nil {
		return err
	}

	imageJob.Status.Skipped = skipped
	if err := r.updateJobStatus(ctx, imageJob); err != nil {
		return err
	}

	podSpecTemplate := imageJob.Spec.JobTemplate.Spec
	for i := range nodeList {
		log := log.WithValues("node", nodeList[i].Name)
		podSpec, err := copyAndFillTemplateSpec(&podSpecTemplate, env, &nodeList[i])
		if err != nil {
			return err
		}

		containerName := podSpec.Containers[0].Name
		nodeName := nodeList[i].Name

		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{},
			Spec:     *podSpec,
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    "eraser-system",
				GenerateName: containerName + "-" + nodeName + "-",
				Labels:       map[string]string{"name": containerName},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(imageJob, imageJob.GroupVersionKind()),
				},
			},
		}

		fitness := checkNodeFitness(pod, &nodeList[i])
		if !fitness {
			log.Info(containerName + " pod does not fit on node, skipping")
			continue
		}

		err = r.Create(ctx, pod)
		if err != nil {
			return err
		}
		log.Info("Started "+containerName+" pod on node", "nodeName", nodeName)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	log.Info("imagejob set up with manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1alpha1.ImageJob{}).
		Complete(r)
}

func getMountPath(runtimeName string) string {
	switch runtimeName {
	case docker:
		return dockerPath
	case containerd:
		return containerdPath
	case crio:
		return crioPath
	default:
		return ""
	}
}

func podsComplete(podList []corev1.Pod) bool {
	for i := range podList {
		if podList[i].Status.Phase == corev1.PodRunning || podList[i].Status.Phase == corev1.PodPending {
			return false
		}
	}
	return true
}

func (r *Reconciler) updateJobStatus(ctx context.Context, imageJob *eraserv1alpha1.ImageJob) error {
	if imageJob.Name != "" {
		if err := r.Status().Update(ctx, imageJob); err != nil {
			return err
		}
	}
	return nil
}

func filterOutSkippedNodes(nodes *corev1.NodeList, skipNodesSelectors []string) ([]corev1.Node, int, error) {
	skipped := 0
	nodeList := make([]corev1.Node, 0, len(nodes.Items))

nodes:
	for i := range nodes.Items {
		log := log.WithValues("node", nodes.Items[i].Name)

		nodeName := nodes.Items[i].Name
		for _, skipNodesSelector := range skipNodesSelectors {
			skipLabels, err := labels.Parse(skipNodesSelector)
			if err != nil {
				return nil, -1, err
			}

			log.V(1).Info("skipLabels", "skipLabels", skipLabels)
			log.V(1).Info("nodeLabels", "nodeLabels", nodes.Items[i].ObjectMeta.Labels)
			if skipLabels.Matches(labels.Set(nodes.Items[i].ObjectMeta.Labels)) {
				log.Info("node will be skipped because it matched the specified labels",
					"nodeName", nodeName,
					"labels", nodes.Items[i].ObjectMeta.Labels,
					"specifiedSelectors", skipNodesSelectors,
				)

				skipped++
				continue nodes
			}
		}

		nodeList = append(nodeList, nodes.Items[i])
	}

	return nodeList, skipped, nil
}

func copyAndFillTemplateSpec(templateSpecTemplate *corev1.PodSpec, env []corev1.EnvVar, node *corev1.Node) (*corev1.PodSpec, error) {
	nodeName := node.Name
	runtime := node.Status.NodeInfo.ContainerRuntimeVersion
	runtimeName := strings.Split(runtime, ":")[0]

	mountPath := getMountPath(runtimeName)
	if mountPath == "" {
		return nil, fmt.Errorf("incompatible runtime on node")
	}

	args := []string{"--runtime=" + runtimeName}
	volumes := []corev1.Volume{
		{Name: runtimeName + "-sock-volume", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: mountPath}}},
	}

	volumeMounts := []corev1.VolumeMount{
		{MountPath: mountPath, Name: runtimeName + "-sock-volume"},
	}

	templateSpec := templateSpecTemplate.DeepCopy()
	image := &templateSpec.Containers[0]

	image.Args = append(args, image.Args...)
	image.VolumeMounts = append(volumeMounts, image.VolumeMounts...)
	image.Env = append(env, image.Env...)
	templateSpec.Volumes = append(volumes, templateSpec.Volumes...)
	templateSpec.NodeName = nodeName

	return templateSpec, nil
}
