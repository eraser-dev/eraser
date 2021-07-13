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

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"

	"k8s.io/kubernetes/pkg/scheduler/algorithm/predicates"
	schedulernodeinfo "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
)

var (
	controllerLog = ctrl.Log.WithName("controllerRuntimeLogger")
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ImageJobReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// ImageJobReconciler reconciles a ImageJob object
type ImageJobReconciler struct {
	client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("imagejob-controller", mgr, controller.Options{
		Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to ImageJob
	err = c.Watch(&source.Kind{Type: &eraserv1alpha1.ImageJob{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ImageJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	controllerLog.Info("imagejob reconcile")

	nodes := (&v1.NodeList{}).Items

	for _, n := range nodes {
		nodeName := n.Name

		pod := &v1.Pod{}
		pod.Namespace = "eraser-system"
		pod.Spec.NodeName = nodeName

		// need?
		removeImage := &v1.Container{}
		removeImage.Image = "ashnam/remove_images"
		pod.Spec.Containers = append(pod.Spec.Containers, *removeImage)

		// check if pod can be scheduled on node
		fitness, err := checkNodeFitness(pod, &n)
		if (err == nil) && (fitness) {
			r.Create(context.TODO(), pod)
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImageJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1alpha1.ImageJob{}).
		Complete(r)
}

// Check if pod can be scheduled on node
// Source: Kruise broadcastjob

// checkNodeFitness runs a set of predicates that select candidate nodes for the job pod;
// the predicates include:
//   - PodFitsHost: checks pod's NodeName against node
//   - PodMatchNodeSelector: checks pod's ImagePullJobNodeSelector and NodeAffinity against node
//   - PodToleratesNodeTaints: exclude tainted node unless pod has specific toleration
//   - CheckNodeUnschedulablePredicate: check if the pod can tolerate node unschedulable
//   - PodFitsResources: checks if a node has sufficient resources, such as cpu, memory, gpu, opaque int resources etc to run a pod.
func checkNodeFitness(pod *v1.Pod, node *v1.Node) (bool, error) {
	nodeInfo := schedulernodeinfo.NewNodeInfo()
	_ = nodeInfo.SetNode(node)

	fit, reasons, err := predicates.PodFitsHost(pod, nil, nodeInfo)
	if err != nil || !fit {
		logPredicateFailedReason(reasons, node)
		return false, err
	}

	fit, reasons, err = predicates.PodMatchNodeSelector(pod, nil, nodeInfo)
	if err != nil || !fit {
		logPredicateFailedReason(reasons, node)
		return false, err
	}

	fit, reasons, err = predicates.PodToleratesNodeTaints(pod, nil, nodeInfo)
	if err != nil || !fit {
		logPredicateFailedReason(reasons, node)
		return false, err
	}

	fit, reasons, err = predicates.CheckNodeUnschedulablePredicate(pod, nil, nodeInfo)
	if err != nil || !fit {
		logPredicateFailedReason(reasons, node)
		return false, err
	}
	fit, reasons, err = predicates.PodFitsResources(pod, nil, nodeInfo)
	if err != nil || !fit {
		logPredicateFailedReason(reasons, node)
		return false, err
	}
	return true, nil
}

func logPredicateFailedReason(reasons []predicates.PredicateFailureReason, node *v1.Node) {
	if len(reasons) == 0 {
		return
	}
	for _, reason := range reasons {
		klog.Errorf("Failed predicate on node %s : %s ", node.Name, reason.GetReason())
	}
}
