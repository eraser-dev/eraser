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

package imagecollector

import (
	"context"
	"flag"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cri-api/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	collectorImage = flag.String("collector-image", "ghcr.io/azure/collector:latest", "collector image")
	log            = logf.Log.WithName("controller").WithValues("process", "imagecollector-controller")
)

// ImageCollectorReconciler reconciles a ImageCollector object
type ImageCollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ImageCollectorReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("imagejob-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch for changes to ImageJob
	err = c.Watch(&source.Kind{Type: &eraserv1alpha1.ImageJob{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to ImageCollector
	err = c.Watch(&source.Kind{Type: &eraserv1alpha1.ImageCollector{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

//+kubebuilder:rbac:groups=eraser.sh.tutorial.kubebuilder.io,resources=imagecollectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh.tutorial.kubebuilder.io,resources=imagecollectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh.tutorial.kubebuilder.io,resources=imagecollectors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageCollector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *ImageCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// periodically create imageJob with collector pods
	job := &eraserv1alpha1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagejob-",
		},
		Spec: eraserv1alpha1.ImageJobSpec{
			JobTemplate: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "collector",
							Image:           *collectorImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					ServiceAccountName: "eraser-controller-manager",
				},
			},
		},
	}

	err := r.Create(ctx, job)
	log.Info("creating imagejob", "job", job.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Second * 5}, nil

	// once job is complete, for each collector in imagecollector list, get list of images
	// store image in deduplicated list
	// create imagecollector-shared crd
	// delete individual imagecollector CRs
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImageCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		Complete(r)
}
