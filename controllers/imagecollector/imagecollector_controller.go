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
	"github.com/Azure/eraser/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	scannerImage           = flag.String("scanner-image", "", "scanner image, empty value disables scan feature")
	collectorImage         = flag.String("collector-image", "", "collector image, empty value disables collect feature")
	log                    = logf.Log.WithName("controller").WithValues("process", "imagecollector-controller")
	repeatPeriod           = flag.Duration("repeat-period", time.Hour*24, "repeat period for collect/scan process")
	deleteScanFailedImages = flag.Bool("delete-scan-failed-images", true, "whether or not to delete images for which scanning has failed")
	scannerArgs            = utils.MultiFlag([]string{})
	collectorArgs          = utils.MultiFlag([]string{})
)

const (
	collectorShared = "imagecollector-shared"
	apiVersion      = "eraser.sh/v1alpha1"
	excludedPath    = "/run/eraser.sh/excluded"
	excludedName    = "excluded"
)

func init() {
	flag.Var(&scannerArgs, "scanner-arg", "An argument to be passed through to the scanner. For example, --scanner-arg=--severity=CRITICAL,HIGH will be passed through to the scanner as --severity=CRITICAL,HIGH. Can be supplied multiple times.")
	flag.Var(&collectorArgs, "collector-arg", "An argument to be passed through to the collector. For example, --collector-arg=--enable-pprof=true will pass through to the collector as --enable-pprof=true. Can be supplied multiple times.")
}

// ImageCollectorReconciler reconciles a ImageCollector object.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func Add(mgr manager.Manager) error {
	if *collectorImage == "" {
		return nil
	}
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

	err = c.Watch(
		&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestForOwner{OwnerType: &eraserv1alpha1.ImageCollector{}, IsController: true},
		predicate.Funcs{
			// Do nothing on Create, Delete, or Generic events
			CreateFunc:  util.NeverOnCreate,
			DeleteFunc:  util.NeverOnDelete,
			GenericFunc: util.NeverOnGeneric,
			UpdateFunc: func(e event.UpdateEvent) bool {
				if job, ok := e.ObjectNew.(*batchv1.Job); ok && job.Status.Succeeded == 1 {
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
//+kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;create;delete;watch

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
	defer log.Info("done reconcile")

	imageCollectorShared := eraserv1alpha1.ImageCollector{
		TypeMeta:   metav1.TypeMeta{Kind: "ImageCollector", APIVersion: apiVersion},
		ObjectMeta: metav1.ObjectMeta{Name: collectorShared},

		Spec: eraserv1alpha1.ImageCollectorSpec{Images: []eraserv1alpha1.Image{}},
	}

	if err := r.Get(ctx, types.NamespacedName{Name: collectorShared}, &imageCollectorShared); err != nil {
		if isNotFound(err) {
			if err := r.Create(ctx, &imageCollectorShared); err != nil {
				log.Info("could not create shared image collector")
				return reconcile.Result{}, err
			}
		} else {
			log.Info("could not get shared image collector")
			return reconcile.Result{}, err
		}
	}

	childImageJobs, err := r.getChildImageJobs(ctx, &imageCollectorShared)
	if err != nil {
		return ctrl.Result{}, err
	}

	switch len(childImageJobs) {
	case 0:
		// If we reach this point, we are in one of two scenarios. Either:
		// (a) Reconcile has been called on a timer, and we want to begin a
		//      collector ImageJob
		// (b) a scan job has just finished, and we need to clean up the scan
		//      job and create an imagelist
		scanJobs, err := r.getChildScanJobs(ctx, &imageCollectorShared)
		if err != nil {
			return ctrl.Result{}, err
		}

		switch len(scanJobs) {
		case 0: // (a) the timer has elapsed; begin a new pipeline
			return r.createImageJob(ctx, req, &imageCollectorShared, collectorArgs)
		case 1: // (b) scanjob has finished; delete and create/update imageList
			err := r.Delete(ctx, &scanJobs[0])
			if err != nil {
				return ctrl.Result{}, err
			}

			// create imagelist (or update if already exists), which will trigger eraser imagejob
			return r.upsertImageList(ctx, &imageCollectorShared)
		default:
			return ctrl.Result{}, fmt.Errorf("more than one scan Jobs are scheduled")
		}
	case 1:
		// an imagejob has just completed; proceed to scanning phase
		// (if enabled) or imagelist creation.
		return r.handleCompletedImageJob(ctx, req, &imageCollectorShared, &childImageJobs[0])
	default:
		return ctrl.Result{}, fmt.Errorf("more than one collector ImageJobs are scheduled")
	}
}

func (r *Reconciler) getChildImageJobs(ctx context.Context, collector *eraserv1alpha1.ImageCollector) ([]eraserv1alpha1.ImageJob, error) {
	imageJobList := &eraserv1alpha1.ImageJobList{}
	if err := r.List(ctx, imageJobList); err != nil {
		log.Info("could not list imagejobs")
		return nil, err
	}

	relevantJobs := util.FilterJobListByOwner(
		imageJobList.Items, metav1.NewControllerRef(collector, schema.GroupVersionKind{
			Group:   "eraser.sh",
			Version: "v1alpha1",
			Kind:    "ImageCollector",
		}),
	)

	return relevantJobs, nil
}

func (r *Reconciler) upsertImageList(ctx context.Context, collector *eraserv1alpha1.ImageCollector) (ctrl.Result, error) {
	imageListItems := make([]string, 0, len(collector.Spec.Images))

	images := collector.Spec.Images
	if !scanDisabled() {
		images = collector.Status.Vulnerable

		if *deleteScanFailedImages {
			images = append(images, collector.Status.Failed...)
		}
	}

	for i := range images {
		img := images[i]
		imageListItems = append(imageListItems, img.Digest)
	}

	imageList := eraserv1alpha1.ImageList{}

	err := r.Get(ctx, types.NamespacedName{Namespace: "", Name: "imagelist"}, &imageList)
	if isNotFound(err) {
		return r.createImageList(ctx, collector, imageListItems)
	}

	// else update
	imageList.Spec.Images = imageListItems

	err = r.Update(ctx, &imageList)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func scanDisabled() bool {
	return *scannerImage == ""
}

func (r *Reconciler) createImageList(ctx context.Context, collector *eraserv1alpha1.ImageCollector, items []string) (ctrl.Result, error) {
	imageList := eraserv1alpha1.ImageList{
		ObjectMeta: metav1.ObjectMeta{
			Name: "imagelist",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					collector,
					schema.GroupVersionKind{
						Group:   "eraser.sh",
						Version: "v1alpha1",
						Kind:    "ImageCollector",
					},
				),
			},
		},
		Spec: eraserv1alpha1.ImageListSpec{
			Images: items,
		},
	}

	err := r.Create(ctx, &imageList)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) getChildScanJobs(ctx context.Context, collector *eraserv1alpha1.ImageCollector) ([]batchv1.Job, error) {
	batchJobList := batchv1.JobList{}
	err := r.List(ctx, &batchJobList, client.InNamespace(utils.GetNamespace()))
	if err != nil {
		return nil, err
	}

	relevantBatchJobs := util.FilterBatchJobListByOwner(
		batchJobList.Items, metav1.NewControllerRef(collector, schema.GroupVersionKind{
			Group:   "eraser.sh",
			Version: "v1alpha1",
			Kind:    "ImageCollector",
		}),
	)

	return relevantBatchJobs, nil
}

func (r *Reconciler) createScanJob(ctx context.Context, collector *eraserv1alpha1.ImageCollector, scannerImage string, args []string) error {
	one := int32(1)
	instanceArgs := make([]string, 0, len(args)+1)
	instanceArgs = append(instanceArgs, "--collector-cr-name="+collector.Name)
	instanceArgs = append(instanceArgs, args...)

	scanJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "eraser-scanner-",
			Namespace:    utils.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(
				collector,
				schema.GroupVersionKind{
					Group:   "eraser.sh",
					Version: "v1alpha1",
					Kind:    "ImageCollector",
				},
			)},
		},
		Spec: batchv1.JobSpec{
			Parallelism: &one,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "scanner-",
					Namespace:    utils.GetNamespace(),
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/os",
												Operator: corev1.NodeSelectorOperator("NotIn"),
												Values:   []string{"windows"},
											},
										},
									},
								},
							},
						},
					},
					ServiceAccountName: "eraser-imagejob-pods",
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "trivy-scanner",
							Image: scannerImage,
							Args:  instanceArgs,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"memory": resource.Quantity{
										Format: resource.Format(*util.ScannerMemRequest),
									},
									"cpu": resource.Quantity{
										Format: resource.Format(*util.ScannerCPURequest),
									},
								},
								Limits: corev1.ResourceList{
									"memory": resource.Quantity{
										Format: resource.Format(*util.ScannerMemLimit),
									},
									"cpu": resource.Quantity{
										Format: resource.Format(*util.ScannerCPULimit),
									},
								},
							},
							SecurityContext: utils.SharedSecurityContext,
						},
					},
				},
			},
		},
	}

	return r.Create(ctx, &scanJob)
}

func isNotFound(err error) bool {
	return err != nil && client.IgnoreNotFound(err) == nil
}

func (r *Reconciler) handleJobDeletion(ctx context.Context, job *eraserv1alpha1.ImageJob) (ctrl.Result, error) {
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

func (r *Reconciler) createImageJob(ctx context.Context, req ctrl.Request, imageCollector *eraserv1alpha1.ImageCollector, args []string) (ctrl.Result, error) {
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
					Volumes: []corev1.Volume{
						{
							Name: excludedName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: excludedName}, Optional: utils.BoolPtr(true)},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "collector",
							Image:           *collectorImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            args,
							VolumeMounts: []corev1.VolumeMount{
								{MountPath: excludedPath, Name: excludedName},
							},
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
							SecurityContext: utils.SharedSecurityContext,
						},
					},
					ServiceAccountName: "eraser-imagejob-pods",
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

func (r *Reconciler) handleCompletedImageJob(ctx context.Context, req ctrl.Request, imageCollectorShared *eraserv1alpha1.ImageCollector, childJob *eraserv1alpha1.ImageJob) (ctrl.Result, error) {
	var err error
	switch phase := childJob.Status.Phase; phase {
	case eraserv1alpha1.PhaseCompleted:
		log.Info("completed phase")
		if childJob.Status.DeleteAfter == nil {
			if res, err := r.updateSharedCRD(ctx, req, imageCollectorShared); err != nil {
				return res, err
			}
			childJob.Status.DeleteAfter = util.After(time.Now(), int64(util.SuccessDel.Seconds()))
			if err := r.Status().Update(ctx, childJob); err != nil {
				log.Info("Could not update Delete After for job " + childJob.Name)
			}
			return ctrl.Result{}, nil
		}

		// if scan is disabled, create/update imagelist to prune since collector job has finished
		if scanDisabled() {
			if res, err := r.upsertImageList(ctx, imageCollectorShared); err != nil {
				return res, err
			}
		} else {
			// else we create a scan job which will update imagelist to start removal with relevant images
			err := r.createScanJob(ctx, imageCollectorShared, *scannerImage, scannerArgs)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		if res, err := r.handleJobDeletion(ctx, childJob); err != nil || res.RequeueAfter > 0 {
			return res, err
		}
	case eraserv1alpha1.PhaseFailed:
		log.Info("failed phase")
		if childJob.Status.DeleteAfter == nil {
			childJob.Status.DeleteAfter = util.After(time.Now(), int64(util.ErrDel.Seconds()))
			if err := r.Status().Update(ctx, childJob); err != nil {
				log.Info("Could not update Delete After for job " + childJob.Name)
			}
			return ctrl.Result{}, nil
		}
		if res, err := r.handleJobDeletion(ctx, childJob); err != nil || res.RequeueAfter > 0 {
			return res, err
		}
	default:
		err = errors.New("should not reach this point for imagejob")
		log.Error(err, "imagejob", childJob)
	}

	return ctrl.Result{RequeueAfter: *repeatPeriod}, err
}
