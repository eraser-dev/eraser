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
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/metric/global"
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
	log                    = logf.Log.WithName("controller").WithValues("process", "imagecollector-controller")
	deleteScanFailedImages = flag.Bool("delete-scan-failed-images", true, "whether or not to delete images for which scanning has failed")
	startTime              time.Time
	ownerLabel             labels.Selector
	exporter               sdkmetric.Exporter
	reader                 sdkmetric.Reader
	provider               *sdkmetric.MeterProvider
)

func init() {
	var err error

	ownerLabelString := fmt.Sprintf("%s=%s", util.ImageJobOwnerLabelKey, ownerLabelValue)
	ownerLabel, err = labels.Parse(ownerLabelString)
	if err != nil {
		panic(err)
	}
}

// ImageCollectorReconciler reconciles a ImageCollector object.
type Reconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	eraserConfig eraserv1.EraserConfig
}

func Add(mgr manager.Manager, cfg eraserv1.EraserConfig) error {

	collCfg := cfg.Components.Collector
	if !collCfg.Enable {
		return nil
	}

	return add(mgr, newReconciler(mgr, cfg))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, cfg eraserv1.EraserConfig) *Reconciler {
	otlpEndpoint := cfg.Manager.OTLPEndpoint
	if otlpEndpoint != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		exporter, reader, provider = metrics.ConfigureMetrics(ctx, log, otlpEndpoint)
		global.SetMeterProvider(provider)
	}

	return &Reconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		eraserConfig: cfg,
	}
}

func add(mgr manager.Manager, r *Reconciler) error {
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

	scheduleCfg := r.eraserConfig.Manager.Scheduling
	delay := time.Duration(scheduleCfg.RepeatInterval)
	if scheduleCfg.BeginImmediately {
		delay = 0 * time.Second
	}

	log.Info("delay", "delay", delay)

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
		return r.createImageJob(ctx, req)
	}

	switch len(imageJobList.Items) {
	case 0:
		// If we reach this point, reconcile has been called on a timer, and we want to begin a
		// collector ImageJob
		return r.createImageJob(ctx, req)
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

func (r *Reconciler) createImageJob(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mgrCfg := r.eraserConfig.Manager
	compCfg := r.eraserConfig.Components

	scanCfg := compCfg.Scanner
	collectorCfg := compCfg.Collector
	eraserCfg := compCfg.Eraser

	scanDisabled := !scanCfg.Enable
	startTime = time.Now()

	eraserImg := *util.EraserImage
	if eraserImg == "" {
		iCfg := eraserCfg.Image
		eraserImg = fmt.Sprintf("%s:%s", iCfg.Repo, iCfg.Tag)
	}

	log.Info("eraserImg", "eraserImg", eraserImg)

	iCfg := collectorCfg.Image
	collectorImg := fmt.Sprintf("%s:%s", iCfg.Repo, iCfg.Tag)

	profileConfig := r.eraserConfig.Manager.Profile
	profileArgs := []string{
		"--enable-pprof=" + strconv.FormatBool(profileConfig.Enable),
		fmt.Sprintf("--pprof-port=%d", profileConfig.Port),
	}

	collArgs := []string{"--scan-disabled=" + strconv.FormatBool(scanDisabled)}
	collArgs = append(collArgs, profileArgs...)

	eraserArgs := append(util.EraserArgs, "--log-level="+logger.GetLevel())
	eraserArgs = append(eraserArgs, profileArgs...)

	jobTemplate := corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					// EmptyDir default
					Name: "shared-data",
				},
				{
					Name: "eraser-manager-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "eraser-manager-config",
							},
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "collector",
					Image:           collectorImg,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            collArgs,
					VolumeMounts: []corev1.VolumeMount{
						{MountPath: "/run/eraser.sh/shared-data", Name: "shared-data"},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"cpu":    collectorCfg.Request.CPU,
							"memory": collectorCfg.Request.Mem,
						},
						Limits: corev1.ResourceList{
							"memory": collectorCfg.Limit.Mem,
						},
					},
				},
				{
					Name:            "eraser",
					Image:           eraserImg,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            eraserArgs,
					VolumeMounts: []corev1.VolumeMount{
						{MountPath: "/run/eraser.sh/shared-data", Name: "shared-data"},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"cpu":    eraserCfg.Request.CPU,
							"memory": eraserCfg.Request.Mem,
						},
						Limits: corev1.ResourceList{
							"memory": eraserCfg.Limit.Mem,
						},
					},
					SecurityContext: utils.SharedSecurityContext,
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: mgrCfg.OTLPEndpoint,
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
		iCfg := scanCfg.Image
		scannerImg := fmt.Sprintf("%s:%s", iCfg.Repo, iCfg.Tag)

		cfgDirname := "/config"
		cfgFilename := filepath.Join(cfgDirname, "controller_manager_config.yaml")
		scannerArgs := []string{fmt.Sprintf("--config=%s", cfgFilename)}
		scannerArgs = append(scannerArgs, profileArgs...)

		scannerContainer := corev1.Container{
			Name:  "trivy-scanner",
			Image: scannerImg,
			Args:  scannerArgs,
			VolumeMounts: []corev1.VolumeMount{
				{MountPath: "/run/eraser.sh/shared-data", Name: "shared-data"},
				{MountPath: cfgDirname, Name: "eraser-manager-config"},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"memory": scanCfg.Request.Mem,
					"cpu":    scanCfg.Request.CPU,
				},
				Limits: corev1.ResourceList{
					"memory": scanCfg.Limit.Mem,
				},
			},
			// env vars for exporting metrics
			Env: []corev1.EnvVar{
				{
					Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
					Value: mgrCfg.OTLPEndpoint,
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
	otlpEndpoint := r.eraserConfig.Manager.OTLPEndpoint
	repeatInterval := time.Duration(r.eraserConfig.Manager.Scheduling.RepeatInterval)

	cleanupCfg := r.eraserConfig.Manager.ImageJob.Cleanup
	successDelay := time.Duration(cleanupCfg.DelayOnSuccess)
	errDelay := time.Duration(cleanupCfg.DelayOnFailure)

	switch phase := childJob.Status.Phase; phase {
	case eraserv1.PhaseCompleted:
		log.Info("completed phase")
		if childJob.Status.DeleteAfter == nil {
			childJob.Status.DeleteAfter = util.After(time.Now(), int64(successDelay.Seconds()))
			if err := r.Status().Update(ctx, childJob); err != nil {
				log.Info("Could not update Delete After for job " + childJob.Name)
			}
			return ctrl.Result{}, nil
		}

		if otlpEndpoint != "" {
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
			childJob.Status.DeleteAfter = util.After(time.Now(), int64(errDelay.Seconds()))
			if err := r.Status().Update(ctx, childJob); err != nil {
				log.Info("Could not update Delete After for job " + childJob.Name)
			}
			return ctrl.Result{}, nil
		}

		if otlpEndpoint != "" {
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

	return ctrl.Result{RequeueAfter: repeatInterval}, err
}
