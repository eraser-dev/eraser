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

package imagelist

import (
	"context"
	"flag"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
)

var (
	log         = logf.Log.WithName("controller").WithValues("process", "imagelist-controller")
	eraserImage = flag.String("eraser-image", "ghcr.io/azure/eraser:latest", "eraser image")
	imageList   = types.NamespacedName{Name: "imagelist"}
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// ImageJobReconciler reconciles a ImageJob object.
type ImageJobReconciler struct {
	client.Client
}

// ImageListReconciler reconciles a ImageList object.
type Reconciler struct {
	client.Client
	scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagelists,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagelists/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagelists/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageList object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Ignore unsupported lists
	if req.NamespacedName != imageList {
		log.Info("Ignoring unsupported imagelist name", "name", req.Name)
		return reconcile.Result{}, nil
	}

	imageList := &eraserv1alpha1.ImageList{}
	err := r.Get(ctx, req.NamespacedName, imageList)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If there is a change in ImageList, start ImageJob to trigger removal
	job := &eraserv1alpha1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagejob-",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(imageList, imageList.GroupVersionKind()),
			},
		},
		Spec: eraserv1alpha1.ImageJobSpec{
			JobTemplate: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "eraser",
							Image:           *eraserImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            []string{"--imagelist=" + req.Name},
						},
					},
					ServiceAccountName: "eraser-controller-manager",
				},
			},
		},
	}

	err = r.Create(ctx, job)
	log.Info("creating imagejob", "job", job.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("imagelist-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &eraserv1alpha1.ImageList{}},
		&handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	return nil
}
