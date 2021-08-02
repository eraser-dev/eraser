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

package imagestatus

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
)

var (
	controllerLog = ctrl.Log.WithName("controllerRuntimeLogger")
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// ImageStatusReconciler reconciles a ImageStatus object
type Reconciler struct {
	client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	/*

		// Create a new controller
		c, err := controller.New("imagestatus-controller", mgr, controller.Options{
			Reconciler: r})
		if err != nil {
			return err
		}


			// Watch for changes to EraserPods
			err = c.Watch(&source.Kind{Type: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "eraser-system"}}}, &handler.EnqueueRequestForObject{})
			if err != nil {
				return err
			} */

	return nil
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagestatuss,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagestatuss/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagestatuss/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageStatus object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	controllerLog.Info("imagestatus reconcile")
	/*
		// if number of eraserpods = number of nodes
		// update status

		podName := req.Name

		pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: podName}}
		err := r.Get(ctx, req.NamespacedName, pod)

		if err != nil {
			controllerLog.Info("err")
			panic(err)
		}

		status := pod.Status

		controllerLog.Info(status.Message, status.Reason, status) */

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1alpha1.ImageStatus{}).
		Complete(r)
}
