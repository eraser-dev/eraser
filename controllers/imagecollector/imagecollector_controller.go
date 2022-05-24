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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	collectorImage = flag.String("collector-image", "ghcr.io/azure/collector:latest", "collector image")
	log            = logf.Log.WithName("controller").WithValues("process", "imagecollector-controller")
)

// ImageCollectorReconciler reconciles a ImageCollector object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	log.Info("add collector controller")
	// Create a new controller
	c, err := controller.New("imagecollector-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	ch := make(chan event.GenericEvent)
	err = c.Watch(&source.Channel{
		Source: ch,
	}, &handler.EnqueueRequestForObject{})

	go func() {
		log.Info("Queueing first ImageCollector reconcile...")
		ch <- event.GenericEvent{
			Object: &eraserv1alpha1.ImageCollector{},
		}
		log.Info("Queued first ImageCollector reconcile")
	}()

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
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Info("ImageCollector Reconcile")
	// periodically create imageJob with collector pods
	// add a label to let imagejob controller know that we dont want to delete the ImageJob so that we can check the status of the job later in reconcile

	// imageList := &eraserv1alpha1.ImageList{}
	// err := r.Get(ctx, req.NamespacedName, imageList)
	// if err != nil {
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }

	// job := &eraserv1alpha1.ImageJob{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		GenerateName: "imagejob-",
	// 		OwnerReferences: []metav1.OwnerReference{
	// 			*metav1.NewControllerRef(imageList, imageList.GroupVersionKind()),
	// 		},
	// 	},
	// 	Spec: eraserv1alpha1.ImageJobSpec{
	// 		JobTemplate: corev1.PodTemplateSpec{
	// 			Spec: corev1.PodSpec{
	// 				RestartPolicy: corev1.RestartPolicyNever,
	// 				Containers: []corev1.Container{
	// 					{
	// 						Name:            "collector",
	// 						Image:           *collectorImage,
	// 						ImagePullPolicy: corev1.PullIfNotPresent,
	// 					},
	// 				},
	// 				ServiceAccountName: "eraser-controller-manager",
	// 			},
	// 		},
	// 	},
	// }

	/*
		ImageCollector controller reads from each imageCollector CR, deduplicates, and removes excluded images/registries (by reading from configmap)
		ToDo - image collector controller decides what to do with images that have digests but no names associated with them
		Image collector controller creates shared imageCollector CR using deduplicated list in spec
	*/

	// err = r.Create(ctx, job)
	// log.Info("creating imagejob", "job", job.Name)
	// if err != nil {
	// 	if errors.IsNotFound(err) {
	// 		return reconcile.Result{}, nil
	// 	}
	// 	return reconcile.Result{}, err
	// }

	return ctrl.Result{RequeueAfter: time.Second * 5}, nil

	// once job is complete, for each collector in imagecollector list, get list of images
	// store image in deduplicated list
	// create imagecollector-shared crd
	// delete individual imagecollector CRs
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1alpha1.ImageCollector{}).
		Complete(r)
}
