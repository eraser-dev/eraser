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
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/metric/global"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eraserv1 "github.com/Azure/eraser/api/v1"
	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/controllers/util"
	"github.com/Azure/eraser/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/eraser/pkg/logger"
	"github.com/Azure/eraser/pkg/metrics"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	ownerLabelValue = "imagecollector"
)

var (
	scannerImage           = flag.String("scanner-image", "", "scanner image, empty value disables scan feature")
	collectorImage         = flag.String("collector-image", "", "collector image, empty value disables collect feature")
	log                    = logf.Log.WithName("controller").WithValues("process", "imagecollector-controller")
	repeatPeriod           = flag.Duration("repeat-period", time.Hour*24, "repeat period for collect/scan process")
	scheduleImmediate      = flag.Bool("schedule-immediate", true, "begin collect/scan process immediately")
	deleteScanFailedImages = flag.Bool("delete-scan-failed-images", true, "whether or not to delete images for which scanning has failed")
	scannerArgs            = utils.MultiFlag([]string{})
	collectorArgs          = utils.MultiFlag([]string{})
	startTime              time.Time
	ownerLabel             labels.Selector
	exporter               sdkmetric.Exporter
	reader                 sdkmetric.Reader
	provider               *sdkmetric.MeterProvider
)

func init() {
	var err error

	flag.Var(&scannerArgs, "scanner-arg", "An argument to be passed through to the scanner. For example, --scanner-arg=--severity=CRITICAL,HIGH will be passed through to the scanner as --severity=CRITICAL,HIGH. Can be supplied multiple times.")
	flag.Var(&collectorArgs, "collector-arg", "An argument to be passed through to the collector. For example, --collector-arg=--enable-pprof=true will pass through to the collector as --enable-pprof=true. Can be supplied multiple times.")

	ownerLabelString := fmt.Sprintf("%s=%s", util.ImageJobOwnerLabelKey, ownerLabelValue)
	ownerLabel, err = labels.Parse(ownerLabelString)
	if err != nil {
		panic(err)
	}
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
	if *util.OtlpEndpoint != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		exporter, reader, provider = metrics.ConfigureMetrics(ctx, log, *util.OtlpEndpoint)
		global.SetMeterProvider(provider)
	}

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
		&source.Kind{Type: &eraserv1.ImageJob{}},
		&handler.EnqueueRequestForObject{}, predicate.Funcs{
			// Do nothing on Create, Delete, or Generic events
			CreateFunc:  util.NeverOnCreate,
			DeleteFunc:  util.NeverOnDelete,
			GenericFunc: util.NeverOnGeneric,
			UpdateFunc: func(e event.UpdateEvent) bool {
				if job, ok := e.ObjectNew.(*eraserv1.ImageJob); ok && util.IsCompletedOrFailed(job.Status.Phase) {
					return ownerLabel.Matches(labels.Set(job.ObjectMeta.Labels))
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

	delay := *repeatPeriod
	if *scheduleImmediate {
		delay = 0 * time.Second
	}

	// runs the provided function after the specified delay
	_ = time.AfterFunc(delay, func() {
		log.Info("Queueing first ImageCollector reconcile...")
		ch <- event.GenericEvent{
			Object: &eraserv1.ImageJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "first-reconcile",
				},
			},
		}
	})

	return nil
}

//+kubebuilder:rbac:groups="",resources=podtemplates,verbs=get;list;watch;create;update;patch;delete

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

	imageJobList := &eraserv1.ImageJobList{}
	if err := r.List(ctx, imageJobList); err != nil {
		log.Info("could not list imagejobs")
		return ctrl.Result{}, err
	}

	if req.Name == "first-reconcile" {
		for idx := range imageJobList.Items {
			if err := r.Delete(ctx, &imageJobList.Items[idx]); err != nil {
				log.Info("error cleaning up previous imagejobs")
				return ctrl.Result{}, err
			}
		}
		return r.createImageJob(ctx, req, collectorArgs)
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

func (r *Reconciler) handleJobDeletion(ctx context.Context, job *eraserv1.ImageJob) (ctrl.Result, error) {
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
	startTime = time.Now()

	jobTemplate := corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					// EmptyDir default
					Name: "shared-data",
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "collector",
					Image:           *collectorImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            append(collectorArgs, "--scan-disabled="+strconv.FormatBool(scanDisabled)),
					VolumeMounts: []corev1.VolumeMount{
						{MountPath: "/run/eraser.sh/shared-data", Name: "shared-data"},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"cpu":    resource.MustParse("7m"),
							"memory": resource.MustParse("25Mi"),
						},
						Limits: corev1.ResourceList{
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
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"cpu":    resource.MustParse("7m"),
							"memory": resource.MustParse("25Mi"),
						},
						Limits: corev1.ResourceList{
							"memory": resource.MustParse("30Mi"),
						},
					},
					SecurityContext: utils.SharedSecurityContext,
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: *util.OtlpEndpoint,
						},
						{
							Name:  "OTEL_SERVICE_NAME",
							Value: "eraser",
						},
					},
				},
			},
			ServiceAccountName: "eraser-imagejob-pods",
		},
	}

	job := &eraserv1alpha1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagejob-",
			Labels: map[string]string{
				util.ImageJobOwnerLabelKey: ownerLabelValue,
			},
		},
	}

	if !scanDisabled {
		deleteFailedString := strconv.FormatBool(*deleteScanFailedImages)
		scanFailedArg := fmt.Sprintf("--delete-scan-failed-images=%s", deleteFailedString)
		scannerArgs = append(scannerArgs, scanFailedArg)

		scannerContainer := corev1.Container{
			Name:  "trivy-scanner",
			Image: *scannerImage,
			Args:  scannerArgs,
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
				},
			},
			// env vars for exporting metrics
			Env: []corev1.EnvVar{
				{
					Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
					Value: *util.OtlpEndpoint,
				},
				{
					Name:  "OTEL_SERVICE_NAME",
					Value: "trivy-scanner",
				},
			},
		}
		jobTemplate.Spec.Containers = append(jobTemplate.Spec.Containers, scannerContainer)
	}

	configmapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configmapList, client.InNamespace(utils.GetNamespace())); err != nil {
		log.Info("Could not get list of configmaps")
		return reconcile.Result{}, err
	}

	exclusionMount, exclusionVolume, err := util.GetExclusionVolume(configmapList)
	if err != nil {
		log.Info("Could not get exclusion mounts and volumes")
		return reconcile.Result{}, err
	}

	for i := range jobTemplate.Spec.Containers {
		jobTemplate.Spec.Containers[i].VolumeMounts = append(jobTemplate.Spec.Containers[i].VolumeMounts, exclusionMount...)
	}

	jobTemplate.Spec.Volumes = append(jobTemplate.Spec.Volumes, exclusionVolume...)

	err = r.Create(ctx, job)
	if err != nil {
		log.Info("Could not create collector ImageJob")
		return reconcile.Result{}, err
	}

	namespace := utils.GetNamespace()
	template := corev1.PodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.GetName(),
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(job, eraserv1alpha1.GroupVersion.WithKind("ImageJob")),
			},
		},
		Template: jobTemplate,
	}
	err = r.Create(ctx, &template)
	if err != nil {
		log.Error(err, "Could not create collector PodTemplate")
		return reconcile.Result{}, err
	}

	log.Info("Successfully created collector ImageJob", "job", job.Name)
	return reconcile.Result{}, nil
}

func (r *Reconciler) handleCompletedImageJob(ctx context.Context, req ctrl.Request, childJob *eraserv1.ImageJob) (ctrl.Result, error) {
	var err error
	switch phase := childJob.Status.Phase; phase {
	case eraserv1.PhaseCompleted:
		log.Info("completed phase")
		if childJob.Status.DeleteAfter == nil {
			childJob.Status.DeleteAfter = util.After(time.Now(), int64(util.SuccessDel.Seconds()))
			if err := r.Status().Update(ctx, childJob); err != nil {
				log.Info("Could not update Delete After for job " + childJob.Name)
			}
			return ctrl.Result{}, nil
		}

		if *util.OtlpEndpoint != "" {
			// record metrics
			if err := metrics.RecordMetricsController(ctx, global.MeterProvider(), float64(time.Since(startTime).Seconds()), int64(childJob.Status.Succeeded), int64(childJob.Status.Failed)); err != nil {
				log.Error(err, "error recording metrics")
			}
			metrics.ExportMetrics(log, exporter, reader, provider)
		}

		if res, err := r.handleJobDeletion(ctx, childJob); err != nil || res.RequeueAfter > 0 {
			return res, err
		}
	case eraserv1.PhaseFailed:
		log.Info("failed phase")
		if childJob.Status.DeleteAfter == nil {
			childJob.Status.DeleteAfter = util.After(time.Now(), int64(util.ErrDel.Seconds()))
			if err := r.Status().Update(ctx, childJob); err != nil {
				log.Info("Could not update Delete After for job " + childJob.Name)
			}
			return ctrl.Result{}, nil
		}

		if *util.OtlpEndpoint != "" {
			// record metrics
			if err := metrics.RecordMetricsController(ctx, global.MeterProvider(), float64(time.Since(startTime).Milliseconds()), int64(childJob.Status.Succeeded), int64(childJob.Status.Failed)); err != nil {
				log.Error(err, "error recording metrics")
			}
			metrics.ExportMetrics(log, exporter, reader, provider)
		}

		if res, err := r.handleJobDeletion(ctx, childJob); err != nil || res.RequeueAfter > 0 {
			return res, err
		}
	default:
		err = errors.New("should not reach this point for imagejob")
		log.Error(err, "imagejob not in completed or failed phase", "imagejob", childJob)
	}

	return ctrl.Result{RequeueAfter: *repeatPeriod}, err
}
