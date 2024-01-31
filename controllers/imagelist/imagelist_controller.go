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
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/metric/global"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/eraser-dev/eraser/api/unversioned/config"
	eraserv1 "github.com/eraser-dev/eraser/api/v1"
	"github.com/eraser-dev/eraser/controllers/util"
	"github.com/eraser-dev/eraser/pkg/logger"
	"github.com/eraser-dev/eraser/pkg/metrics"
	eraserUtils "github.com/eraser-dev/eraser/pkg/utils"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	imgListPath     = "/run/eraser.sh/imagelist"
	ownerLabelValue = "imagelist-controller"
)

var (
	log        = logf.Log.WithName("controller").WithValues("process", "imagelist-controller")
	imageList  = types.NamespacedName{Name: "imagelist"}
	ownerLabel labels.Selector
	startTime  time.Time
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

func Add(mgr manager.Manager, cfg *config.Manager) error {
	r, err := newReconciler(mgr, cfg)
	if err != nil {
		return err
	}

	return add(mgr, r)
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, cfg *config.Manager) (reconcile.Reconciler, error) {
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
		scheme:       mgr.GetScheme(),
		eraserConfig: cfg,
	}

	return rec, nil
}

// ImageJobReconciler reconciles a ImageJob object.
type ImageJobReconciler struct {
	client.Client
}

// ImageListReconciler reconciles a ImageList object.
type Reconciler struct {
	client.Client
	scheme       *runtime.Scheme
	eraserConfig *config.Manager
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagelists,verbs=get;list;watch
//+kubebuilder:rbac:groups="",namespace="system",resources=podtemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagelists/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",namespace="system",resources=pods,verbs=get;list;watch;update;create;delete

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

	imageList := eraserv1.ImageList{}
	err := r.Get(ctx, req.NamespacedName, &imageList)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	jobList := eraserv1.ImageJobList{}
	err = r.List(ctx, &jobList)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	items := util.FilterJobListByOwner(jobList.Items, metav1.NewControllerRef(&imageList, imageList.GroupVersionKind()))

	switch len(items) {
	case 0:
		return r.handleImageListEvent(ctx, &imageList)
	case 1:
		job := items[0]

		// If we got here because of a completed ImageJob:
		if util.IsCompletedOrFailed(job.Status.Phase) {
			return r.handleJobListEvent(ctx, &imageList, &job)
		}

		// If we got here due to an update to the ImageList, and there is an ImageJob already running,
		// keep requeueing it until that job is completed.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("there are multiple child imagejobs running")
	}
}

func (r *Reconciler) handleJobListEvent(ctx context.Context, imageList *eraserv1.ImageList, job *eraserv1.ImageJob) (ctrl.Result, error) {
	phase := job.Status.Phase
	if phase == eraserv1.PhaseCompleted || phase == eraserv1.PhaseFailed {
		err := r.handleJobCompletion(ctx, imageList, job)
		if err != nil {
			return ctrl.Result{}, err
		}

		eraserConfig, err := r.eraserConfig.Read()
		if err != nil {
			return ctrl.Result{}, err
		}

		cleanupCfg := eraserConfig.Manager.ImageJob.Cleanup
		successDelay := time.Duration(cleanupCfg.DelayOnSuccess)
		errDelay := time.Duration(cleanupCfg.DelayOnFailure)

		if job.Status.DeleteAfter == nil {
			if job.Status.Phase == eraserv1.PhaseCompleted {
				job.Status.DeleteAfter = util.After(time.Now(), int64(successDelay.Seconds()))
			} else if job.Status.Phase == eraserv1.PhaseFailed {
				job.Status.DeleteAfter = util.After(time.Now(), int64(errDelay.Seconds()))
			}

			if err := r.Status().Update(ctx, job); err != nil {
				log.Info("Could not update Delete After for job " + job.Name)
			}
			return ctrl.Result{}, nil
		}

		otlpEndpoint := eraserConfig.Manager.OTLPEndpoint
		if otlpEndpoint != "" {
			// record metrics
			if err := metrics.RecordMetricsController(ctx, global.MeterProvider(), float64(time.Since(startTime).Seconds()), int64(job.Status.Succeeded), int64(job.Status.Failed)); err != nil {
				log.Error(err, "error recording metrics")
			}
			metrics.ExportMetrics(log, exporter, reader)
		}

		return r.handleJobDeletion(ctx, job)
	}

	return ctrl.Result{}, fmt.Errorf("unexpected job phase: '%s'", job.Status.Phase)
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

	return ctrl.Result{}, nil
}

func (r *Reconciler) handleImageListEvent(ctx context.Context, imageList *eraserv1.ImageList) (ctrl.Result, error) {
	imgListJSON, err := json.Marshal(imageList.Spec.Images)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("marshal image list: %w", err)
	}

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagelist-",
			Namespace:    eraserUtils.GetNamespace(),
		},
		Immutable: eraserUtils.BoolPtr(true),
		Data:      map[string]string{"images": string(imgListJSON)},
	}
	if err := r.Create(ctx, &configMap); err != nil {
		return ctrl.Result{}, fmt.Errorf("create configmap: %w", err)
	}

	configName := configMap.Name
	args := []string{
		"--imagelist=" + filepath.Join(imgListPath, "images"),
		"--log-level=" + logger.GetLevel(),
	}

	eraserConfig, err := r.eraserConfig.Read()
	if err != nil {
		return ctrl.Result{}, err
	}

	eraserContainerCfg := eraserConfig.Components.Remover
	imageCfg := eraserContainerCfg.Image
	image := fmt.Sprintf("%s:%s", imageCfg.Repo, imageCfg.Tag)

	pullSecrets := []corev1.LocalObjectReference{}
	for _, secret := range eraserConfig.Manager.PullSecrets {
		pullSecrets = append(pullSecrets, corev1.LocalObjectReference{Name: secret})
	}

	jobTemplate := corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: configName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: configName}},
					},
				},
			},
			ImagePullSecrets:  pullSecrets,
			RestartPolicy:     corev1.RestartPolicyNever,
			PriorityClassName: eraserConfig.Manager.PriorityClassName,
			Containers: []corev1.Container{
				{
					Name:            "remover",
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            args,
					VolumeMounts: []corev1.VolumeMount{
						{MountPath: imgListPath, Name: configName},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"cpu":    eraserContainerCfg.Request.CPU,
							"memory": eraserContainerCfg.Request.Mem,
						},
						Limits: corev1.ResourceList{
							"memory": eraserContainerCfg.Limit.Mem,
						},
					},
					SecurityContext: eraserUtils.SharedSecurityContext,
					// env vars for exporting metrics
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: eraserConfig.Manager.OTLPEndpoint,
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

	job := &eraserv1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagejob-",
			Labels: map[string]string{
				util.ImageJobOwnerLabelKey: ownerLabelValue,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(imageList, eraserv1.GroupVersion.WithKind("ImageList")),
			},
		},
	}

	configmapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configmapList); err != nil {
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
	startTime = time.Now()
	log.Info("creating imagejob", "job", job.Name)

	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// get manager pod with label control-plane=controller-manager
	podList := corev1.PodList{}
	if err = r.List(ctx, &podList, client.InNamespace(eraserUtils.GetNamespace()), client.MatchingLabels{"control-plane": "controller-manager"}); err != nil {
		log.Info("Unable to list controller-manager pod")
	}
	if len(podList.Items) != 1 {
		log.Info("Incorrect number of controller-manager pods", "number of pods", len(podList.Items))
	}
	managerPod := &podList.Items[0]

	template := corev1.PodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.GetName(),
			Namespace: eraserUtils.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(managerPod, managerPod.GroupVersionKind()),
			},
		},
		Template: jobTemplate,
	}

	err = r.Create(ctx, &template)
	if err != nil {
		return reconcile.Result{}, err
	}

	configMap.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(job, eraserv1.GroupVersion.WithKind("ImageJob"))}
	err = r.Update(ctx, &configMap)
	if err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) handleJobCompletion(ctx context.Context, imageList *eraserv1.ImageList, job *eraserv1.ImageJob) error {
	now := metav1.Now()

	imageList.Status.Success = int64(job.Status.Succeeded)
	imageList.Status.Failed = int64(job.Status.Failed)
	imageList.Status.Skipped = int64(job.Status.Skipped)
	imageList.Status.Timestamp = &now

	err := r.Status().Update(ctx, imageList)
	if err != nil {
		return err
	}

	return nil
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("imagelist-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &eraserv1.ImageList{}},
		&handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &eraserv1.ImageJob{}},
		&handler.EnqueueRequestForOwner{OwnerType: &eraserv1.ImageList{}, IsController: true},
		predicate.Funcs{
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

	return nil
}
