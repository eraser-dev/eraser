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
	"errors"
	"flag"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/controllers/util"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	collectorImage = flag.String("collector-image", "ghcr.io/azure/collector:latest", "collector image")
	log            = logf.Log.WithName("controller").WithValues("process", "imagecollector-controller")
)

const (
	repeatPeriod    = time.Minute * 10
	collectorShared = "imagecollector-shared"
	apiVersion      = "eraser.sh/v1alpha1"
)

// ImageCollectorReconciler reconciles a ImageCollector object.
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

	err = c.Watch(
		&source.Kind{Type: &eraserv1alpha1.ImageJob{}},
		&handler.EnqueueRequestForOwner{OwnerType: &eraserv1alpha1.ImageCollector{}, IsController: true},
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

	ch := make(chan event.GenericEvent)
	err = c.Watch(&source.Channel{
		Source: ch,
	}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	go func() {
		log.Info("Queueing first ImageCollector reconcile...")
		ch <- event.GenericEvent{
			Object: &eraserv1alpha1.ImageCollector{},
		}
		log.Info("Queued first ImageCollector reconcile")
	}()

	return nil
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagecollectors,verbs=get;list;watch;create;update;patch;delete
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

	imageCollectorShared := &eraserv1alpha1.ImageCollector{
		TypeMeta:   metav1.TypeMeta{Kind: "ImageCollector", APIVersion: apiVersion},
		ObjectMeta: metav1.ObjectMeta{Name: collectorShared},

		Spec: eraserv1alpha1.ImageCollectorSpec{Images: []eraserv1alpha1.Image{}},
	}

	if err := r.Get(ctx, types.NamespacedName{Name: collectorShared, Namespace: "default"}, imageCollectorShared); err != nil {
		if isNotFound(err) {
			if err := r.Create(ctx, imageCollectorShared); err != nil {
				log.Info("could not create shared image collector")
				return reconcile.Result{}, err
			}
		} else {
			log.Info("could not get shared image collector")
			return reconcile.Result{}, err
		}
	}

	imageJobList := &eraserv1alpha1.ImageJobList{}
	if err := r.List(ctx, imageJobList); err != nil {
		log.Info("could not list imagejobs")
		return reconcile.Result{}, err
	}

	relevantJobs := util.FilterJobListByOwner(
		imageJobList.Items, metav1.NewControllerRef(imageCollectorShared, schema.GroupVersionKind{
			Group:   "eraser.sh",
			Version: "v1alpha1",
			Kind:    "ImageCollector",
		}),
	)

	if len(relevantJobs) > 1 {
		return reconcile.Result{}, fmt.Errorf("More than one collector ImageJob scheduled")
	}
	if len(relevantJobs) == 0 {
		if res, err := r.createImageJob(ctx, req, imageCollectorShared); err != nil {
			return res, err
		}
		return ctrl.Result{RequeueAfter: repeatPeriod}, nil
	}

	// else length is 1, so check job phase
	switch phase := relevantJobs[0].Status.Phase; phase {
	case eraserv1alpha1.PhaseCompleted:
		log.Info("completed phase")
		if relevantJobs[0].Status.DeleteAfter == nil {
			if res, err := r.updateSharedCRD(ctx, req, imageCollectorShared); err != nil {
				return res, err
			}
			relevantJobs[0].Status.DeleteAfter = util.After(time.Now(), *util.SuccessDelDelaySeconds)
			if err := r.Status().Update(ctx, &relevantJobs[0]); err != nil {
				log.Info("Could not update Delete After for job " + relevantJobs[0].Name)
			}
		}
		if res, err := r.handleJobDeletion(ctx, &relevantJobs[0]); err != nil || res.RequeueAfter > 0 {
			return res, err
		}
	case eraserv1alpha1.PhaseFailed:
		log.Info("failed phase")
		if relevantJobs[0].Status.DeleteAfter == nil {
			relevantJobs[0].Status.DeleteAfter = util.After(time.Now(), *util.ErrDelDelaySeconds)
			if err := r.Status().Update(ctx, &relevantJobs[0]); err != nil {
				log.Info("Could not update Delete After for job " + relevantJobs[0].Name)
			}
		}
		if res, err := r.handleJobDeletion(ctx, &relevantJobs[0]); err != nil || res.RequeueAfter > 0 {
			return res, err
		}
	default:
		log.Error(errors.New("should not reach this point for imagejob"), "imagejob: ", relevantJobs[0])
	}

	log.Info("done reconcile")

	return ctrl.Result{RequeueAfter: repeatPeriod}, nil
}

func isNotFound(err error) bool {
	if err != nil && client.IgnoreNotFound(err) == nil {
		return true
	}
	return false
}

func (r *Reconciler) handleJobDeletion(ctx context.Context, job *eraserv1alpha1.ImageJob) (ctrl.Result, error) {
	log.Info("start job deletion")
	if job.Status.DeleteAfter == nil {
		log.Info("delete after is nil")
		return ctrl.Result{}, nil
	}

	until := time.Until(job.Status.DeleteAfter.Time)
	if until > 0 {
		log.Info("Delaying imagejob delete", "job", job.Name, "deleteAter", job.Status.DeleteAfter)
		return ctrl.Result{RequeueAfter: until}, nil
	}

	log.Info("Deleting imagejob", "job", job.Name)
	err := r.Delete(ctx, job)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("end job deletion")
	return ctrl.Result{}, nil
}

func (r *Reconciler) createImageJob(ctx context.Context, req ctrl.Request, imageCollector *eraserv1alpha1.ImageCollector) (ctrl.Result, error) {
	job := &eraserv1alpha1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagejob-",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(imageCollector, schema.GroupVersionKind{Group: "eraser.sh", Version: "v1alpha1", Kind: "ImageCollector"}),
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
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("7m"),
									"memory": resource.MustParse("25Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("8m"),
									"memory": resource.MustParse("30Mi"),
								},
							},
						},
					},
					ServiceAccountName: "eraser-controller-manager",
				},
			},
		},
	}

	err := r.Create(ctx, job)
	if err != nil {
		log.Info("Could not create collector ImageJob")
		return reconcile.Result{}, err
	}

	log.Info("Successfully created collector ImageJob", "job", job.Name)
	return reconcile.Result{}, nil
}

func (r *Reconciler) updateSharedCRD(ctx context.Context, req ctrl.Request, imageCollector *eraserv1alpha1.ImageCollector) (ctrl.Result, error) {
	imageCollectorList := &eraserv1alpha1.ImageCollectorList{}
	if err := r.List(ctx, imageCollectorList); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	items := imageCollectorList.Items

	// store images in map to remove duplicates
	// map with key: sha id, value: name of image
	idToNameMap := make(map[string]string)

	for i := range items {
		if items[i].Name != collectorShared {
			temp := items[i].Spec.Images
			for _, img := range temp {
				idToNameMap[img.Digest] = img.Name
			}
		}
	}

	var combined []eraserv1alpha1.Image

	for key, value := range idToNameMap {
		combined = append(combined, eraserv1alpha1.Image{Digest: key, Name: value})
	}

	imageCollector.Spec = eraserv1alpha1.ImageCollectorSpec{Images: combined}

	if err := r.Update(ctx, imageCollector); err != nil {
		log.Info("Could not update imageCollector spec")
		return reconcile.Result{}, err
	}

	if res, err := r.deleteNodeCRS(ctx, items); err != nil {
		return res, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) deleteNodeCRS(ctx context.Context, items []eraserv1alpha1.ImageCollector) (ctrl.Result, error) {
	// delete individual image collector CRs
	for i := range items {
		if items[i].Name != collectorShared {
			if err := r.Delete(ctx, &items[i]); err != nil {
				log.Info("Delete", "Could not delete image collector", items[i].Name)
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}
