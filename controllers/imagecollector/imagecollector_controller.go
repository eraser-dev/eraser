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
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
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

	"github.com/Azure/eraser/pkg/logger"
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
	excludedPath = "/run/eraser.sh/excluded"
	excludedName = "excluded"
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
		&handler.EnqueueRequestForObject{}, predicate.Funcs{
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
			Object: &eraserv1alpha1.ImageJob{},
		}
		log.Info("Queued first ImageCollector reconcile")
	}()

	return nil
}

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

	imageJobList := &eraserv1alpha1.ImageJobList{}
	if err := r.List(ctx, imageJobList); err != nil {
		log.Info("could not list imagejobs")
		return ctrl.Result{}, err
	}

	switch len(imageJobList.Items) {
	case 0:
		// If we reach this point, reconcile has been called on a timer, and we want to begin a
		// collector ImageJob
		return r.createImageJob(ctx, req, collectorArgs)
	case 1:
		// an imagejob has just completed; proceed to imagelist creation.
		return r.handleCompletedImageJob(ctx, req, &imageJobList.Items[0])
	default:
		return ctrl.Result{}, fmt.Errorf("more than one collector ImageJobs are scheduled")
	}
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

func (r *Reconciler) createImageJob(ctx context.Context, req ctrl.Request, argsCollector []string) (ctrl.Result, error) {
	scanDisabled := *scannerImage == ""

	job := &eraserv1alpha1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagejob-",
		},
		Spec: eraserv1alpha1.ImageJobSpec{
			JobTemplate: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: excludedName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: excludedName}, Optional: boolPtr(true)},
							},
						},
						{
							// EmptyDir default
							Name: "shared-data",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					// init container creates named pipes
					InitContainers: []corev1.Container{
						{
							Name:            "init-collector-pod",
							Image:           "docker.io/library/busybox:latest", // eraser image?
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{MountPath: "/run/eraser.sh/shared-data", Name: "shared-data"},
							},
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{"mkfifo /run/eraser.sh/shared-data/collectScan; mkfifo /run/eraser.sh/shared-data/scanErase; chown -R 65532:65532 /run/eraser.sh/shared-data/"},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "collector",
							Image:           *collectorImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            append(collectorArgs, "--scan-disabled="+strconv.FormatBool(scanDisabled)),
							VolumeMounts: []corev1.VolumeMount{
								{MountPath: excludedPath, Name: excludedName},
								{MountPath: "/run/eraser.sh/shared-data", Name: "shared-data"},
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
						},
						{
							Name:            "eraser",
							Image:           *util.EraserImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            append(util.EraserArgs, "--log-level="+logger.GetLevel()),
							VolumeMounts: []corev1.VolumeMount{
								{MountPath: "/run/eraser.sh/shared-data", Name: "shared-data"},
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
						},
					},
					ServiceAccountName: "eraser-imagejob-pods",
				},
			},
		},
	}

	if !scanDisabled {
		scannerContainer := corev1.Container{
			Name:  "trivy-scanner",
			Image: *scannerImage,
			Args:  append(scannerArgs, "delete-scan-failed-images="+strconv.FormatBool(*deleteScanFailedImages)),
			VolumeMounts: []corev1.VolumeMount{
				{MountPath: "/run/eraser.sh/shared-data", Name: "shared-data"},
			},
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
		}
		job.Spec.JobTemplate.Spec.Containers = append(job.Spec.JobTemplate.Spec.Containers, scannerContainer)
	}

	err := r.Create(ctx, job)
	if err != nil {
		log.Info("Could not create collector ImageJob")
		return reconcile.Result{}, err
	}

	log.Info("Successfully created collector ImageJob", "job", job.Name)
	return reconcile.Result{}, nil
}

func boolPtr(b bool) *bool {
	return &b
}

func (r *Reconciler) handleCompletedImageJob(ctx context.Context, req ctrl.Request, childJob *eraserv1alpha1.ImageJob) (ctrl.Result, error) {
	var err error
	switch phase := childJob.Status.Phase; phase {
	case eraserv1alpha1.PhaseCompleted:
		log.Info("completed phase")
		if childJob.Status.DeleteAfter == nil {
			childJob.Status.DeleteAfter = util.After(time.Now(), int64(util.SuccessDel.Seconds()))
			if err := r.Status().Update(ctx, childJob); err != nil {
				log.Info("Could not update Delete After for job " + childJob.Name)
			}
			return ctrl.Result{}, nil
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
