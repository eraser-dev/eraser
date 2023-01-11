package util

import (
	"flag"
	"time"

	eraserv1 "github.com/Azure/eraser/api/v1"
	"github.com/Azure/eraser/api/v1/config"
	"github.com/Azure/eraser/pkg/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var (
	EraserImage = flag.String("eraser-image", "", "eraser image")
	EraserArgs  = utils.MultiFlag([]string{})

	EraserOptions = config.Default()
)

const (
	ImageJobOwnerLabelKey = "eraser.sh/job-owner"

	exclusionLabel = "eraser.sh/exclude.list=true"

	EnvVarContainerdNamespaceKey   = "CONTAINERD_NAMESPACE"
	EnvVarContainerdNamespaceValue = "k8s.io"
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

func IsCompletedOrFailed(p eraserv1.JobPhase) bool {
	return (p == eraserv1.PhaseCompleted || p == eraserv1.PhaseFailed)
}

func FilterJobListByOwner(jobs []eraserv1.ImageJob, owner *metav1.OwnerReference) []eraserv1.ImageJob {
	ret := []eraserv1.ImageJob{}

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
