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

	"github.com/eraser-dev/eraser/api/unversioned/config"
	eraserv1 "github.com/eraser-dev/eraser/api/v1"
	eraserv1alpha1 "github.com/eraser-dev/eraser/api/v1alpha1"
	"github.com/eraser-dev/eraser/controllers/util"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/eraser-dev/eraser/pkg/logger"
	"github.com/eraser-dev/eraser/pkg/metrics"
	eraserUtils "github.com/eraser-dev/eraser/pkg/utils"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ownerLabelValue  = "imagecollector"
	configVolumeName = "eraser-config"
)

var (
	log        = logf.Log.WithName("controller").WithValues("process", "imagecollector-controller")
	startTime  time.Time
	ownerLabel labels.Selector
	exporter   sdkmetric.Exporter
	reader     sdkmetric.Reader
	provider   *sdkmetric.MeterProvider
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
	eraserConfig *config.Manager
}

func Add(mgr manager.Manager, cfg *config.Manager) error {
	c, err := cfg.Read()
	if err != nil {
		return err
	}

	collCfg := c.Components.Collector
	if !collCfg.Enabled {
		// don't add controller, but don't throw an error either
		return nil
	}

	r, err := newReconciler(mgr, cfg)
	if err != nil {
		return err
	}

	return add(mgr, r)
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, cfg *config.Manager) (*Reconciler, error) {
	c, err := cfg.Read()
	if err != nil {
		return nil, err
	}

	otlpEndpoint := c.Manager.OTLPEndpoint
	if otlpEndpoint != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		exporter, reader, provider = metrics.ConfigureMetrics(ctx, log, otlpEndpoint)
		global.SetMeterProvider(provider)
	}

	rec := &Reconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		eraserConfig: cfg,
	}

	return rec, nil
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

	eraserConfig, err := r.eraserConfig.Read()
	if err != nil {
		return err
	}

	scheduleCfg := eraserConfig.Manager.Scheduling
	delay := time.Duration(scheduleCfg.RepeatInterval)
	if scheduleCfg.BeginImmediately {
		delay = 0 * time.Second
	}

	log.V(1).Info("delay", "delay", delay)

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

//+kubebuilder:rbac:groups=eraser.sh,resources=imagelists,verbs=get;list;watch
//+kubebuilder:rbac:groups="",namespace="system",resources=podtemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagelists/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",namespace="system",resources=pods,verbs=get;list;watch;update;create;delete

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
		return r.createImageJob(ctx)
	}

	switch len(imageJobList.Items) {
	case 0:
		// If we reach this point, reconcile has been called on a timer, and we want to begin a
		// collector ImageJob
		return r.createImageJob(ctx)
	case 1:
		// an imagejob has just completed; proceed to imagelist creation.
		return r.handleCompletedImageJob(ctx, &imageJobList.Items[0])
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

	template := corev1.PodTemplate{}
	if err := r.Get(ctx,
		types.NamespacedName{
			Namespace: eraserUtils.GetNamespace(),
			Name:      job.GetName(),
		},
		&template,
	); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Deleting pod template", "template", template.Name)
	if err := r.Delete(ctx, &template); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("end job deletion")
	return ctrl.Result{}, nil
}

func (r *Reconciler) createImageJob(ctx context.Context) (ctrl.Result, error) {
	eraserConfig, err := r.eraserConfig.Read()
	if err != nil {
		return ctrl.Result{}, err
	}

	mgrCfg := eraserConfig.Manager
	compCfg := eraserConfig.Components

	scanCfg := compCfg.Scanner
	collectorCfg := compCfg.Collector
	eraserCfg := compCfg.Remover

	scanDisabled := !scanCfg.Enabled
	startTime = time.Now()

	removerImg := *util.RemoverImage
	if removerImg == "" {
		iCfg := eraserCfg.Image
		removerImg = fmt.Sprintf("%s:%s", iCfg.Repo, iCfg.Tag)
	}

	log.V(1).Info("removerImg", "removerImg", removerImg)

	iCfg := collectorCfg.Image
	collectorImg := fmt.Sprintf("%s:%s", iCfg.Repo, iCfg.Tag)

	profileConfig := eraserConfig.Manager.Profile
	profileArgs := []string{
		"--enable-pprof=" + strconv.FormatBool(profileConfig.Enabled),
		fmt.Sprintf("--pprof-port=%d", profileConfig.Port),
	}

	collArgs := []string{"--scan-disabled=" + strconv.FormatBool(scanDisabled)}
	collArgs = append(collArgs, profileArgs...)

	removerArgs := []string{"--log-level=" + logger.GetLevel()}
	removerArgs = append(removerArgs, profileArgs...)

	pullSecrets := []corev1.LocalObjectReference{}
	for _, secret := range eraserConfig.Manager.PullSecrets {
		pullSecrets = append(pullSecrets, corev1.LocalObjectReference{Name: secret})
	}

	jobTemplate := corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					// EmptyDir default
					Name: "shared-data",
				},
				{
					Name: configVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: util.EraserConfigmapName,
							},
						},
					},
				},
			},
			ImagePullSecrets:  pullSecrets,
			RestartPolicy:     corev1.RestartPolicyNever,
			PriorityClassName: eraserConfig.Manager.PriorityClassName,
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
					Name:            "remover",
					Image:           removerImg,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            removerArgs,
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
					SecurityContext: eraserUtils.SharedSecurityContext,
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: mgrCfg.OTLPEndpoint,
						},
						{
							Name:  "OTEL_SERVICE_NAME",
							Value: "remover",
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
				{MountPath: cfgDirname, Name: configVolumeName},
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
				{
					Name:  "ERASER_RUNTIME_NAME",
					Value: string(mgrCfg.Runtime.Name),
				},
			},
		}
		jobTemplate.Spec.Containers = append(jobTemplate.Spec.Containers, scannerContainer)
	}

	configmapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configmapList, client.InNamespace(eraserUtils.GetNamespace())); err != nil {
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

	// get manager pod with label control-plane=controller-manager
	podList := corev1.PodList{}
	if err := r.List(ctx, &podList, client.InNamespace(eraserUtils.GetNamespace()), client.MatchingLabels{"control-plane": "controller-manager"}); err != nil {
		log.Info("Unable to list controller-manager pod")
	}
	if len(podList.Items) != 1 {
		log.Info("Incorrect number of controller-manager pods", "number of pods", len(podList.Items))
	}
	managerPod := &podList.Items[0]

	namespace := eraserUtils.GetNamespace()
	template := corev1.PodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.GetName(),
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(managerPod, managerPod.GroupVersionKind()),
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

func (r *Reconciler) handleCompletedImageJob(ctx context.Context, childJob *eraserv1.ImageJob) (ctrl.Result, error) {
	var err error
	var timeRemaining time.Duration
	eraserConfig, err := r.eraserConfig.Read()
	if err != nil {
		return ctrl.Result{}, err
	}

	otlpEndpoint := eraserConfig.Manager.OTLPEndpoint
	repeatInterval := time.Duration(eraserConfig.Manager.Scheduling.RepeatInterval)

	cleanupCfg := eraserConfig.Manager.ImageJob.Cleanup
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
			metrics.ExportMetrics(log, exporter, reader)
		}

		timeRemaining = repeatInterval - successDelay
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
			metrics.ExportMetrics(log, exporter, reader)
		}

		timeRemaining = repeatInterval - errDelay
		if res, err := r.handleJobDeletion(ctx, childJob); err != nil || res.RequeueAfter > 0 {
			return res, err
		}
	default:
		err = errors.New("should not reach this point for imagejob")
		log.Error(err, "imagejob not in completed or failed phase", "imagejob", childJob)
	}

	if timeRemaining <= 0 {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{RequeueAfter: timeRemaining}, err
}
