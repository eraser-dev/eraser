package util

import (
	"context"
	"flag"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/pkg/utils"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var (
	SuccessDel = flag.Duration("job-cleanup-on-success-delay", 0, "Duration to delay job deletion after successful runs. 0 means no delay. Default unit is ns.")
	ErrDel     = flag.Duration("job-cleanup-on-error-delay", time.Hour*24, "Duration to delay job deletion after errored runs. 0 means no delay. Default unit is ns.")

	ScannerCPURequest = flag.String("scanner-cpu-request", "1000m", "minimum CPU request for scanner pods spawned by the eraser manager")
	ScannerCPULimit   = flag.String("scanner-cpu-limit", "1500m", "limit on CPU usage for scanner pods spawned by the eraser manager")
	ScannerMemRequest = flag.String("scanner-mem-request", "500Mi", "minimum memory request for scanner pods spawned by the eraser manager")
	ScannerMemLimit   = flag.String("scanner-mem-limit", "2Gi", "limit on memory usage for scanner pods spawned by the eraser manager")

	EraserImage  = flag.String("eraser-image", "ghcr.io/azure/eraser:latest", "eraser image")
	EraserArgs   = utils.MultiFlag([]string{})
	OtlpEndpoint = flag.String("otlp-endpoint", "", "otel exporter otlp endpoint")
)

const (
	exclusionLabel = "eraser.sh/exclude.list=true"
)

func init() {
	flag.Var(&EraserArgs, "eraser-arg", "An argument to be passed through to the eraser. For example, --eraser-arg=--enable-pprof=true will pass through to the eraser as --enable-pprof=true. Can be supplied multiple times.")
}

func NeverOnCreate(_ event.CreateEvent) bool {
	return false
}

func NeverOnDelete(_ event.DeleteEvent) bool {
	return false
}

func NeverOnGeneric(_ event.GenericEvent) bool {
	return false
}

func NeverOnUpdate(_ event.UpdateEvent) bool {
	return false
}

func AlwaysOnCreate(_ event.CreateEvent) bool {
	return true
}

func AlwaysOnDelete(_ event.DeleteEvent) bool {
	return true
}

func AlwaysOnGeneric(_ event.GenericEvent) bool {
	return true
}

func AlwaysOnUpdate(_ event.UpdateEvent) bool {
	return true
}

func IsCompletedOrFailed(p eraserv1alpha1.JobPhase) bool {
	return (p == eraserv1alpha1.PhaseCompleted || p == eraserv1alpha1.PhaseFailed)
}

func FilterJobListByOwner(jobs []eraserv1alpha1.ImageJob, owner *metav1.OwnerReference) []eraserv1alpha1.ImageJob {
	ret := []eraserv1alpha1.ImageJob{}

	for i := range jobs {
		job := jobs[i]

		for j := range job.OwnerReferences {
			or := job.OwnerReferences[j]

			if or.UID == owner.UID {
				ret = append(ret, job)
				break // inner
			}
		}
	}

	return ret
}

func FilterBatchJobListByOwner(jobs []batchv1.Job, owner *metav1.OwnerReference) []batchv1.Job {
	ret := []batchv1.Job{}

	for i := range jobs {
		job := jobs[i]

		for j := range job.OwnerReferences {
			or := job.OwnerReferences[j]

			if or.UID == owner.UID {
				ret = append(ret, job)
				break // inner
			}
		}
	}

	return ret
}

func After(t time.Time, seconds int64) *metav1.Time {
	newT := metav1.NewTime(t.Add(time.Duration(seconds) * time.Second))
	return &newT
}

func GetExclusionVolume(configmapList *corev1.ConfigMapList) ([]corev1.VolumeMount, []corev1.Volume, error) {
	var exclusionMount []corev1.VolumeMount
	var exclusionVolume []corev1.Volume

	selector, err := labels.Parse(exclusionLabel)
	if err != nil {
		return nil, nil, err
	}

	for i := range configmapList.Items {
		cm := configmapList.Items[i]
		if selector.Matches(labels.Set(cm.ObjectMeta.Labels)) {
			exclusionMount = append(exclusionMount, corev1.VolumeMount{MountPath: "exclude-" + cm.Name, Name: cm.Name})
			exclusionVolume = append(exclusionVolume, corev1.Volume{
				Name: cm.Name,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name}},
				},
			})
		}
	}

	return exclusionMount, exclusionVolume, nil
}

func ConfigureMetrics(ctx context.Context, log logr.Logger) (sdkmetric.Exporter, sdkmetric.Reader, *sdkmetric.MeterProvider) {
	exporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure(), otlpmetrichttp.WithEndpoint(*OtlpEndpoint))
	if err != nil {
		log.Error(err, "error initializing exporter")
		return nil, nil, nil
	}

	reader := sdkmetric.NewPeriodicReader(exporter)
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	return exporter, reader, provider
}

func ExportMetrics(log logr.Logger, exporter sdkmetric.Exporter, reader sdkmetric.Reader, provider *sdkmetric.MeterProvider) {
	ctxB := context.Background()

	m, err := reader.Collect(ctxB)
	if err != nil {
		log.Error(err, "failed to collect metrics")
		return
	}
	if err := exporter.Export(ctxB, m); err != nil {
		log.Error(err, "failed to export metrics")
	}
	if err := provider.Shutdown(ctxB); err != nil {
		log.Error(err, "error during metric shutdown", err)
	}
}
