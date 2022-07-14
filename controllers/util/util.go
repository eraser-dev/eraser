package util

import (
	"flag"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var (
	SuccessDel = flag.Duration("job-cleanup-on-success-delay", 0, "Duration to delay job deletion after successful runs. 0 means no delay. Default unit is ns.")
	ErrDel     = flag.Duration("job-cleanup-on-error-delay", time.Hour*24, "Duration to delay job deletion after errored runs. 0 means no delay. Default unit is ns.")

	ScannerCPURequest = flag.String("scanner-cpu-request", "1000m", "minimum CPU request for scanner pods spawned by the eraser manager")
	ScannerCPULimit   = flag.String("scanner-cpu-limit", "1500m", "limit on CPU usage for scanner pods spawned by the eraser manager")
	ScannerMemRequest = flag.String("scanner-mem-request", "500Mi", "minimum memory request for scanner pods spawned by the eraser manager")
	ScannerMemLimit   = flag.String("scanner-mem-limit", "2Gi", "limit on memory usage for scanner pods spawned by the eraser manager")
)

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
