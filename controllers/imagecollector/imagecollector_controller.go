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
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cri-api/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/controllers/util"

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

	err = c.Watch(
		&source.Kind{Type: &eraserv1alpha1.ImageJob{}},
		&handler.EnqueueRequestForOwner{OwnerType: &eraserv1alpha1.ImageList{}, IsController: true},
		predicate.Funcs{
			// Do nothing on Create, Delete, or Generic events
			CreateFunc:  util.NeverOnCreate,
			DeleteFunc:  util.NeverOnDelete,
			GenericFunc: util.NeverOnGeneric,
			UpdateFunc: func(e event.UpdateEvent) bool {
				if job, ok := e.ObjectNew.(*eraserv1alpha1.ImageJob); ok && util.IsCompletedOrFailed(job.Status.Phase) {
					return true
				}

				return false
			},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagecollectors,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=eraser.sh,resources=imagecollectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagecollectors/finalizers,verbs=update

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

	collector := eraserv1alpha1.ImageCollector{
		ObjectMeta: metav1.ObjectMeta{Name: "initiator"},
		Spec: eraserv1alpha1.ImageCollectorSpec{
			Images: []eraserv1alpha1.Image{},
		},
	}

	log.Info("WE ARE HERE 1")
	err := r.Client.Get(ctx, types.NamespacedName{Name: "initiator"}, &collector)
	if err != nil {
		log.Info("WE ARE HERE 2")
		if client.IgnoreNotFound(err) == nil {
			log.Info("WE ARE HERE 3")
			err = r.Client.Create(ctx, &collector)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			log.Info("WE ARE HERE 4")
			return ctrl.Result{}, err
		}
	}

	log.Info("WE ARE HERE 5")
	jobList := eraserv1alpha1.ImageJobList{}
	err = r.Client.List(ctx, &jobList)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	imageJobs := util.FilterJobListByOwner(jobList.Items, metav1.NewControllerRef(&collector, schema.GroupVersionKind{
		Group:   "eraser.sh",
		Version: "v1alpha1",
		Kind:    "ImageCollector",
	}))

	switch len(imageJobs) {
	case 0:
		break
	case 1:
		job := imageJobs[0]
		// keep requeueing it until that job is completed.

		collectors := eraserv1alpha1.ImageCollectorList{}
		err = r.Client.List(ctx, &collectors)
		if err != nil {
			return reconcile.Result{}, err
		}

		m := make(map[string]string)
		for i := range collectors.Items {
			collector := collectors.Items[i]

			for j := range collector.Spec.Images {
				img := collector.Spec.Images[j]
				m[img.Digest] = img.Name
			}
		}

		newCollector := eraserv1alpha1.ImageCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "collector-cr",
			},
		}

		for digest, name := range m {
			newCollector.Spec.Images = append(newCollector.Spec.Images, eraserv1alpha1.Image{
				Name:   name,
				Digest: digest,
			})
		}

		err = r.Client.DeleteAllOf(ctx, &collector)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Client.Delete(ctx, &job)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Client.Create(ctx, &newCollector)
		if err != nil {
			return ctrl.Result{}, err
		}

		one := int32(1)

		scanJob := batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bill",
				Namespace: "eraser-system",
			},

			Spec: batchv1.JobSpec{
				Parallelism: &one,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "bill-",
						Namespace:    "eraser-system",
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "eraser-controller-manager",
						RestartPolicy:      corev1.RestartPolicyNever,
						Containers: []corev1.Container{
							{
								Name:    "bill",
								Image:   "ghcr.io/azure/eraser-trivy-scanner:v0.1.0",
								Command: []string{"/scanner"},
								Args:    []string{"--cache-dir=/home/nonroot/"},
							},
						},
					},
				},
			},
		}

		err = r.Client.Create(ctx, &scanJob)
		if err != nil {
			return ctrl.Result{}, err
		}

		// return ctrl.Result{RequeueAfter: time.Minute}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("there are multiple child imagejobs running")
	}

	log.Info("WE ARE HERE")
	job := &eraserv1alpha1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagejob-",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&collector, collector.GroupVersionKind()),
			},
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

	/*
		ImageCollector controller reads from each imageCollector CR, deduplicates, and removes excluded images/registries (by reading from configmap)
		ToDo - image collector controller decides what to do with images that have digests but no names associated with them
		Image collector controller creates shared imageCollector CR using deduplicated list in spec
	*/

	err = r.Create(ctx, job)
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
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1alpha1.ImageCollector{}).
		Complete(r)
}
