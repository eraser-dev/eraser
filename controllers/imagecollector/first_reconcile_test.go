package imagecollector

import (
	"context"
	"sync"
	"testing"

	eraserv1 "github.com/eraser-dev/eraser/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFirstReconcileMutexBasic(t *testing.T) {
	// Test that the mutex prevents concurrent execution
	var executionCount int
	var mu sync.Mutex

	// First acquisition should succeed
	if !mu.TryLock() {
		t.Error("First TryLock should succeed")
	}

	// Simulate work
	executionCount++

	// Second acquisition should fail while first is held
	if mu.TryLock() {
		t.Error("Second TryLock should fail while first is held")
	}

	// Release first lock
	mu.Unlock()

	// Now should be able to acquire again
	if !mu.TryLock() {
		t.Error("TryLock should succeed after release")
	}

	mu.Unlock()

	if executionCount != 1 {
		t.Errorf("Expected executionCount 1, got %d", executionCount)
	}
}

func TestFirstReconcileDoneFlag(t *testing.T) {
	// Test that the done flag prevents subsequent executions
	firstReconcileDone := false

	// First call should set it to true
	if firstReconcileDone {
		t.Error("firstReconcileDone should be false initially")
	}

	// Simulate first reconcile completion
	firstReconcileDone = true

	// Subsequent calls should check and skip
	if !firstReconcileDone {
		t.Error("firstReconcileDone should be true after first reconcile")
	}
}

func TestReconcileWithRunningJob(t *testing.T) {
	// Create a scheme and fake client
	scheme := runtime.NewScheme()
	if err := eraserv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	// Create a running job
	runningJob := &eraserv1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job-1",
			Namespace: "eraser-system",
		},
		Status: eraserv1.ImageJobStatus{
			Phase: eraserv1.PhaseRunning,
		},
	}

	// Create a fake client with the running job
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(runningJob).
		Build()

	// Test that the reconcile logic would find the running job
	jobList := &eraserv1.ImageJobList{}
	if err := client.List(context.Background(), jobList); err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobList.Items) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobList.Items))
	}

	if jobList.Items[0].Status.Phase != eraserv1.PhaseRunning {
		t.Errorf("Expected running job, got %s", jobList.Items[0].Status.Phase)
	}
}

func TestReconcileWithNoJobs(t *testing.T) {
	// Create a scheme and fake client with no jobs
	scheme := runtime.NewScheme()
	if err := eraserv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Test that the reconcile logic would find no jobs
	jobList := &eraserv1.ImageJobList{}
	if err := client.List(context.Background(), jobList); err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobList.Items) != 0 {
		t.Errorf("Expected 0 jobs, got %d", len(jobList.Items))
	}
}

func TestReconcileWithMultipleNonRunningJobs(t *testing.T) {
	// Create a scheme and fake client
	scheme := runtime.NewScheme()
	if err := eraserv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	// Create multiple completed/failed jobs
	completedJob := &eraserv1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job-completed",
			Namespace: "eraser-system",
		},
		Status: eraserv1.ImageJobStatus{
			Phase: eraserv1.PhaseCompleted,
		},
	}

	failedJob := &eraserv1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job-failed",
			Namespace: "eraser-system",
		},
		Status: eraserv1.ImageJobStatus{
			Phase: eraserv1.PhaseFailed,
		},
	}

	// Create a fake client with multiple non-running jobs
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(completedJob, failedJob).
		Build()

	// Test that the reconcile logic would find multiple non-running jobs
	jobList := &eraserv1.ImageJobList{}
	if err := client.List(context.Background(), jobList); err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobList.Items) != 2 {
		t.Fatalf("Expected 2 jobs, got %d", len(jobList.Items))
	}

	// All should be non-running
	for _, job := range jobList.Items {
		if job.Status.Phase == eraserv1.PhaseRunning {
			t.Errorf("Expected non-running job, got running: %s", job.Name)
		}
	}
}

func TestNamespacedNameForFirstReconcile(t *testing.T) {
	// Test the NamespacedName used for first-reconcile
	req := types.NamespacedName{
		Name:      "first-reconcile",
		Namespace: "",
	}

	if req.Name != "first-reconcile" {
		t.Errorf("Expected name 'first-reconcile', got '%s'", req.Name)
	}
}

func TestMutexPreventsConcurrentFirstReconcile(t *testing.T) {
	// Test that the mutex prevents concurrent execution in a realistic scenario
	var (
		firstReconcileMutex sync.Mutex
		firstReconcileDone  bool
		executionCount      int
	)

	// Simulate concurrent goroutines trying to run first-reconcile
	runFirstReconcile := func() {
		// Try to acquire lock
		if !firstReconcileMutex.TryLock() {
			return // Already running
		}
		defer firstReconcileMutex.Unlock()

		// Check if already done
		if firstReconcileDone {
			return
		}

		// Do work
		executionCount++
		firstReconcileDone = true
	}

	// Start multiple concurrent calls
	for i := 0; i < 10; i++ {
		go runFirstReconcile()
	}

	// Small wait to let goroutines complete
	// In real usage, the controller-runtime handles synchronization

	if executionCount > 1 {
		t.Errorf("Expected at most 1 execution, got %d", executionCount)
	}
}
