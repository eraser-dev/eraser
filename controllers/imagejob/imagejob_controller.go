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
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	"sigs.k8s.io/kind/pkg/errors"

	"github.com/eraser-dev/eraser/api/unversioned"
	"github.com/eraser-dev/eraser/api/unversioned/config"
	eraserv1 "github.com/eraser-dev/eraser/api/v1"
	controllerUtils "github.com/eraser-dev/eraser/controllers/util"
	eraserUtils "github.com/eraser-dev/eraser/pkg/utils"
)

const (
	defaultFilterLabel   = "eraser.sh/cleanup.filter"
	windowsFilterLabel   = "kubernetes.io/os=windows"
	imageJobTypeLabelKey = "eraser.sh/type"
	collectorJobType     = "collector"
	manualJobType        = "manual"
	removerContainer     = "remover"
	managerLabelValue    = "controller-manager"
	managerLabelKey      = "control-plane"
)

var log = logf.Log.WithName("controller").WithValues("process", "imagejob-controller")

var defaultTolerations = []corev1.Toleration{
	{
		Operator: corev1.TolerationOpExists,
	},
}

func Add(mgr manager.Manager, cfg *config.Manager) error {
	return add(mgr, newReconciler(mgr, cfg))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, cfg *config.Manager) reconcile.Reconciler {
	rec := &Reconciler{
		Client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		eraserConfig: cfg,
	}

	return rec
}

// ImageJobReconciler reconciles a ImageJob object.
type Reconciler struct {
	client.Client
	scheme       *runtime.Scheme
	eraserConfig *config.Manager
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
	err = c.Watch(&source.Kind{Type: &eraserv1.ImageJob{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if job, ok := e.ObjectNew.(*eraserv1.ImageJob); ok && controllerUtils.IsCompletedOrFailed(job.Status.Phase) {
				return false // handled by Owning controller
			}

			return true
		},
		CreateFunc:  controllerUtils.AlwaysOnCreate,
		GenericFunc: controllerUtils.NeverOnGeneric,
		DeleteFunc:  controllerUtils.NeverOnDelete,
	})
	if err != nil {
		return err
	}

	// Watch for changes to pods created by ImageJob (eraser pods)
	err = c.Watch(
		&source.Kind{
			Type: &corev1.Pod{},
		},
		&handler.EnqueueRequestForOwner{
			OwnerType:    &corev1.PodTemplate{},
			IsController: true,
		},
		predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return e.Object.GetNamespace() == eraserUtils.GetNamespace()
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectNew.GetNamespace() == eraserUtils.GetNamespace()
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return e.Object.GetNamespace() == eraserUtils.GetNamespace()
			},
		},
	)
	if err != nil {
		return err
	}

	// watch for changes to imagejob podTemplate (owned by controller manager pod)
	err = c.Watch(
		&source.Kind{
			Type: &corev1.PodTemplate{},
		},
		&handler.EnqueueRequestForOwner{
			OwnerType:    &corev1.Pod{},
			IsController: true,
		},
		predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				ownerLabels, ok := e.Object.GetLabels()[managerLabelKey]
				return ok && ownerLabels == managerLabelValue
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				ownerLabels, ok := e.ObjectNew.GetLabels()[managerLabelKey]
				return ok && ownerLabels == managerLabelValue
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				ownerLabels, ok := e.Object.GetLabels()[managerLabelKey]
				return ok && ownerLabels == managerLabelValue
			},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func checkNodeFitness(pod *corev1.Pod, node *corev1.Node) bool {
	nodeInfo := framework.NewNodeInfo()
	nodeInfo.SetNode(node)

	insufficientResource := noderesources.Fits(pod, nodeInfo)

	if len(insufficientResource) != 0 {
		log.Error(fmt.Errorf("pod %v in namespace %v does not fit in node %v", pod.Name, pod.Namespace, node.Name), "insufficient resource")
		return false
	}

	return true
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups="",namespace="system",resources=podtemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",namespace="system",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

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
	imageJob := &eraserv1.ImageJob{}
	if err := r.Get(ctx, req.NamespacedName, imageJob); err != nil {
		imageJob.Status.Phase = eraserv1.PhaseFailed
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
	case eraserv1.PhaseRunning:
		if err := r.handleRunningJob(ctx, imageJob); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconcile running: %w", err)
		}
	case eraserv1.PhaseCompleted, eraserv1.PhaseFailed:
		break // this is handled by the Owning controller
	default:
		return ctrl.Result{}, fmt.Errorf("reconcile: unexpected imagejob phase: %s", imageJob.Status.Phase)
	}

	return ctrl.Result{}, nil
}

func podListOptions(jobTemplate *corev1.PodTemplate) client.ListOptions {
	var set map[string]string

	if jobTemplate.Template.Spec.Containers[0].Name == removerContainer {
		set = map[string]string{imageJobTypeLabelKey: manualJobType}
	} else {
		set = map[string]string{imageJobTypeLabelKey: collectorJobType}
	}

	return client.ListOptions{
		Namespace:     eraserUtils.GetNamespace(),
		LabelSelector: labels.SelectorFromSet(set),
	}
}

func (r *Reconciler) handleRunningJob(ctx context.Context, imageJob *eraserv1.ImageJob) error {
	// get eraser pods
	podList := &corev1.PodList{}

	template := corev1.PodTemplate{}
	namespace := eraserUtils.GetNamespace()

	err := r.Get(ctx, types.NamespacedName{
		Name:      imageJob.GetName(),
		Namespace: namespace,
	}, &template)
	if err != nil {
		imageJob.Status = eraserv1.ImageJobStatus{
			Phase:       eraserv1.PhaseFailed,
			DeleteAfter: controllerUtils.After(time.Now(), 1),
		}
		return r.updateJobStatus(ctx, imageJob)
	}

	listOpts := podListOptions(&template)
	err = r.List(ctx, podList, &listOpts)
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

	imageJob.Status = eraserv1.ImageJobStatus{
		Desired:   imageJob.Status.Desired,
		Succeeded: success,
		Skipped:   skipped,
		Failed:    failed,
		Phase:     eraserv1.PhaseCompleted,
	}

	successAndSkipped := success + skipped

	eraserConfig, err := r.eraserConfig.Read()
	if err != nil {
		return err
	}

	managerConfig := eraserConfig.Manager
	successRatio := managerConfig.ImageJob.SuccessRatio

	if float64(successAndSkipped/imageJob.Status.Desired) < successRatio {
		log.Info(
			"Marking job as failed",
			"success ratio", successRatio,
			"actual ratio", success/imageJob.Status.Desired,
		)
		imageJob.Status.Phase = eraserv1.PhaseFailed
	}

	return r.updateJobStatus(ctx, imageJob)
}

func (r *Reconciler) handleNewJob(ctx context.Context, imageJob *eraserv1.ImageJob) error {
	nodes := &corev1.NodeList{}
	err := r.List(ctx, nodes)
	if err != nil {
		return err
	}

	template := corev1.PodTemplate{}
	err = r.Get(ctx,
		types.NamespacedName{
			Namespace: eraserUtils.GetNamespace(),
			Name:      imageJob.GetName(),
		},
		&template,
	)
	if err != nil {
		return err
	}

	imageJob.Status = eraserv1.ImageJobStatus{
		Desired:   len(nodes.Items),
		Succeeded: 0,
		Skipped:   0, // placeholder, updated below
		Failed:    0,
		Phase:     eraserv1.PhaseRunning,
	}

	skipped := 0
	var nodeList []corev1.Node

	log := log.WithValues("job", imageJob.Name)

	env := []corev1.EnvVar{
		{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
	}

	eraserConfig, err := r.eraserConfig.Read()
	if err != nil {
		return err
	}
	log.V(1).Info("configuration used", "manager", eraserConfig.Manager, "components", eraserConfig.Components)

	filterOpts := eraserConfig.Manager.NodeFilter
	if !slices.Contains(filterOpts.Selectors, defaultFilterLabel) {
		filterOpts.Selectors = append(filterOpts.Selectors, defaultFilterLabel)
	}

	switch filterOpts.Type {
	case "exclude":
		nodeList, skipped, err = filterOutSkippedNodes(nodes, filterOpts.Selectors)
		if err != nil {
			return err
		}
	case "include":
		nodeList, skipped, err = selectIncludedNodes(nodes, filterOpts.Selectors)
		if err != nil {
			return err
		}
	default:
		return errors.Errorf("invalid node filter option")
	}

	imageJob.Status.Skipped = skipped
	if err := r.updateJobStatus(ctx, imageJob); err != nil {
		return err
	}

	var namespacedNames []types.NamespacedName
	podSpecTemplate := template.Template.Spec
	for i := range nodeList {
		log := log.WithValues("node", nodeList[i].Name)
		podSpec, err := copyAndFillTemplateSpec(&podSpecTemplate, env, &nodeList[i], &eraserConfig.Manager.Runtime)
		if err != nil {
			return err
		}

		containerName := podSpec.Containers[0].Name
		nodeName := nodeList[i].Name

		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{},
			Spec:     *podSpec,
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    eraserUtils.GetNamespace(),
				GenerateName: "eraser-" + nodeName + "-",
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(&template, template.GroupVersionKind()),
				},
			},
		}

		pod.Labels = map[string]string{}

		for k, v := range eraserConfig.Manager.AdditionalPodLabels {
			pod.Labels[k] = v
		}

		if containerName == removerContainer {
			pod.Labels[imageJobTypeLabelKey] = manualJobType
		} else {
			pod.Labels[imageJobTypeLabelKey] = collectorJobType
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
		namespacedNames = append(namespacedNames, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace})
	}

	for _, namespacedName := range namespacedNames {
		if err := wait.PollImmediate(time.Nanosecond, time.Minute*5, r.isPodReady(ctx, namespacedName)); err != nil {
			log.Error(err, "timed out waiting for pod to leave pending state", "pod NamespacedName", namespacedName)
		}
	}

	return nil
}

func (r *Reconciler) isPodReady(ctx context.Context, namespacedName types.NamespacedName) wait.ConditionFunc {
	return func() (bool, error) {
		currentPod := &corev1.Pod{}

		if err := r.Get(ctx, namespacedName, currentPod); err != nil {
			return false, client.IgnoreNotFound(err)
		}

		return currentPod.Status.Phase != corev1.PodPhase(corev1.PodPending), nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	log.Info("imagejob set up with manager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1.ImageJob{}).
		Complete(r)
}

func podsComplete(podList []corev1.Pod) bool {
	for i := range podList {
		if podList[i].Status.Phase == corev1.PodRunning || podList[i].Status.Phase == corev1.PodPending {
			return containersFailed(&podList[i])
		}
	}
	return true
}

func containersFailed(pod *corev1.Pod) bool {
	statuses := pod.Status.ContainerStatuses
	for i := range statuses {
		if statuses[i].State.Terminated != nil && statuses[i].State.Terminated.ExitCode != 0 {
			return true
		}
	}
	return false
}

func (r *Reconciler) updateJobStatus(ctx context.Context, imageJob *eraserv1.ImageJob) error {
	if imageJob.Name != "" {
		if err := r.Status().Update(ctx, imageJob); err != nil {
			return err
		}
	}
	return nil
}

func selectIncludedNodes(nodes *corev1.NodeList, includeNodesSelectors []string) ([]corev1.Node, int, error) {
	skipped := 0
	nodeList := make([]corev1.Node, 0, len(nodes.Items))

nodes:
	for i := range nodes.Items {
		log := log.WithValues("node", nodes.Items[i].Name)
		skipped++
		nodeName := nodes.Items[i].Name
		for _, includeNodesSelectors := range includeNodesSelectors {
			includedLabels, err := labels.Parse(includeNodesSelectors)
			if err != nil {
				return nil, -1, err
			}

			log.V(1).Info("includedLabels", "includedLabels", includedLabels)
			log.V(1).Info("nodeLabels", "nodeLabels", nodes.Items[i].ObjectMeta.Labels)
			if includedLabels.Matches(labels.Set(nodes.Items[i].ObjectMeta.Labels)) {
				log.Info("node is included because it matched the specified labels",
					"nodeName", nodeName,
					"labels", nodes.Items[i].ObjectMeta.Labels,
					"specifiedSelectors", includeNodesSelectors,
				)

				nodeList = append(nodeList, nodes.Items[i])
				skipped--
				continue nodes
			}
		}
	}

	return nodeList, skipped, nil
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

func copyAndFillTemplateSpec(templateSpecTemplate *corev1.PodSpec, env []corev1.EnvVar, node *corev1.Node, runtimeSpec *unversioned.RuntimeSpec) (*corev1.PodSpec, error) {
	nodeName := node.Name

	u, err := url.Parse(runtimeSpec.Address)
	if err != nil {
		return nil, err
	}

	volumes := []corev1.Volume{
		{Name: "runtime-sock-volume", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: u.Path}}},
	}

	volumeMounts := []corev1.VolumeMount{
		{MountPath: controllerUtils.CRIPath, Name: "runtime-sock-volume"},
	}

	templateSpec := templateSpecTemplate.DeepCopy()
	templateSpec.Tolerations = defaultTolerations

	eraserImg := &templateSpec.Containers[0]
	eraserImg.VolumeMounts = append(eraserImg.VolumeMounts, volumeMounts...)
	eraserImg.Env = append(eraserImg.Env, env...)

	if len(templateSpec.Containers) > 1 {
		collectorImg := &templateSpec.Containers[1]
		collectorImg.VolumeMounts = append(collectorImg.VolumeMounts, volumeMounts...)
		collectorImg.Env = append(collectorImg.Env, env...)
	}

	if len(templateSpec.Containers) > 2 {
		scannerImg := &templateSpec.Containers[2]
		scannerImg.VolumeMounts = append(scannerImg.VolumeMounts, volumeMounts...)
		scannerImg.Env = append(scannerImg.Env,
			corev1.EnvVar{
				Name:  controllerUtils.EnvVarContainerdNamespaceKey,
				Value: controllerUtils.EnvVarContainerdNamespaceValue,
			},
		)
		scannerImg.Env = append(scannerImg.Env, env...)
	}

	secrets := os.Getenv("ERASER_PULL_SECRET_NAMES")
	if secrets != "" {
		for _, secret := range strings.Split(secrets, ",") {
			templateSpec.ImagePullSecrets = append(templateSpec.ImagePullSecrets, corev1.LocalObjectReference{Name: secret})
		}
	}

	templateSpec.Volumes = append(volumes, templateSpec.Volumes...)
	templateSpec.NodeName = nodeName

	return templateSpec, nil
}
