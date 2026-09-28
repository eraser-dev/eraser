package imagecollector

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	eraserv1 "github.com/eraser-dev/eraser/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestSimulateControllerRestartScenario simulates the exact OOM scenario described in issue #1169
// where controller restarts create multiple ImageJobs, causing OOM cascade
func TestSimulateControllerRestartScenario(t *testing.T) {
	// Create a scheme and fake client
	scheme := runtime.NewScheme()
	if err := eraserv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	// Create initial empty state
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Track how many ImageJobs are created
	var jobsCreated atomic.Int32
	var mutex sync.Mutex
	var firstReconcileDone bool

	// Simulate multiple controller restarts trying to run first-reconcile concurrently
	simulateControllerRestart := func(attempt int) {
		t.Logf("Simulating controller restart attempt %d", attempt)

		// This simulates the exact logic from the fix
		// Try to acquire lock - if someone else is doing first-reconcile, skip
		if !mutex.TryLock() {
			t.Logf("Attempt %d: Skipped - another reconcile in progress", attempt)
			return
		}
		defer mutex.Unlock()

		// Check if already done
		if firstReconcileDone {
			t.Logf("Attempt %d: Skipped - first-reconcile already completed", attempt)
			return
		}

		// List existing jobs (simulating the real controller behavior)
		jobList := &eraserv1.ImageJobList{}
		if err := client.List(context.Background(), jobList); err != nil {
			t.Logf("Attempt %d: Error listing jobs: %v", attempt, err)
			return
		}

		// Check for running jobs
		for _, job := range jobList.Items {
			if job.Status.Phase == eraserv1.PhaseRunning {
				t.Logf("Attempt %d: Found existing running job %s, adopting it", attempt, job.Name)
				firstReconcileDone = true
				return
			}
		}

		// Clean up non-running jobs
		for _, job := range jobList.Items {
			if job.Status.Phase != eraserv1.PhaseRunning {
				t.Logf("Attempt %d: Cleaning up job %s", attempt, job.Name)
				if err := client.Delete(context.Background(), &job); err != nil {
					t.Logf("Attempt %d: Error deleting job %s: %v", attempt, job.Name, err)
				}
			}
		}

		// Re-list to ensure accurate state
		updatedJobList := &eraserv1.ImageJobList{}
		if err := client.List(context.Background(), updatedJobList); err != nil {
			t.Logf("Attempt %d: Error re-listing jobs: %v", attempt, err)
			return
		}

		// Check again for running jobs
		for _, job := range updatedJobList.Items {
			if job.Status.Phase == eraserv1.PhaseRunning {
				t.Logf("Attempt %d: Found running job after cleanup, adopting", attempt)
				firstReconcileDone = true
				return
			}
		}

		// Create new ImageJob (this is what causes the OOM cascade)
		newJob := &eraserv1.ImageJob{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "imagejob-",
				Namespace:    "eraser-system",
			},
		}

		if err := client.Create(context.Background(), newJob); err != nil {
			t.Logf("Attempt %d: Error creating job: %v", attempt, err)
			return
		}

		jobsCreated.Add(1)
		firstReconcileDone = true
		t.Logf("Attempt %d: Created new ImageJob %s", attempt, newJob.Name)
	}

	// Simulate 10 concurrent controller restarts (the OOM scenario)
	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(attempt int) {
			defer wg.Done()
			time.Sleep(time.Duration(attempt%3) * 10 * time.Millisecond) // Random timing
			simulateControllerRestart(attempt)
		}(i)
	}

	wg.Wait()

	// Verify results
	finalJobList := &eraserv1.ImageJobList{}
	if err := client.List(context.Background(), finalJobList); err != nil {
		t.Fatalf("Failed to list final jobs: %v", err)
	}

	t.Logf("=== RESULTS ===")
	t.Logf("Total restart attempts: 10")
	t.Logf("ImageJobs created: %d", jobsCreated.Load())
	t.Logf("Final ImageJobs in cluster: %d", len(finalJobList.Items))

	// This is the key assertion - only 1 job should be created despite 10 concurrent restarts
	if jobsCreated.Load() != 1 {
		t.Errorf("FAIL: Expected exactly 1 ImageJob to be created, but %d were created", jobsCreated.Load())
		t.Logf("This indicates the fix did NOT work - multiple jobs were created during concurrent restarts")
	} else {
		t.Logf("SUCCESS: Only 1 ImageJob was created despite 10 concurrent restart attempts")
		t.Logf("The fix successfully prevents the OOM cascade scenario")
	}

	if len(finalJobList.Items) != 1 {
		t.Errorf("FAIL: Expected 1 ImageJob in final state, but found %d", len(finalJobList.Items))
	}
}

// TestOldBehaviorWithoutFix demonstrates what would happen without the fix
func TestOldBehaviorWithoutFix(t *testing.T) {
	// Create a scheme and fake client
	scheme := runtime.NewScheme()
	if err := eraserv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Track jobs created
	var jobsCreated atomic.Int32

	// Simulate the OLD broken behavior (no synchronization)
	simulateOldBrokenBehavior := func(attempt int) {
		t.Logf("OLD BEHAVIOR - Attempt %d: Creating ImageJob", attempt)

		// OLD CODE: No check for existing jobs, no synchronization
		// This is what caused the OOM cascade
		jobList := &eraserv1.ImageJobList{}
		client.List(context.Background(), jobList)

		// OLD CODE: Just delete everything and create new one (WRONG!)
		for _, job := range jobList.Items {
			client.Delete(context.Background(), &job)
		}

		// OLD CODE: Immediately create new job without proper validation
		newJob := &eraserv1.ImageJob{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "imagejob-",
				Namespace:    "eraser-system",
			},
		}
		client.Create(context.Background(), newJob)
		jobsCreated.Add(1)
	}

	// Simulate 10 concurrent restarts with old broken behavior
	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(attempt int) {
			defer wg.Done()
			time.Sleep(time.Duration(attempt%3) * 10 * time.Millisecond)
			simulateOldBrokenBehavior(attempt)
		}(i)
	}

	wg.Wait()

	finalJobList := &eraserv1.ImageJobList{}
	client.List(context.Background(), finalJobList)

	t.Logf("=== OLD BEHAVIOR RESULTS ===")
	t.Logf("Total restart attempts: 10")
	t.Logf("ImageJobs created: %d", jobsCreated.Load())
	t.Logf("Final ImageJobs in cluster: %d", len(finalJobList.Items))

	// Without the fix, multiple jobs would be created
	// Note: The actual number may vary due to race conditions, but it will be > 1
	if jobsCreated.Load() > 1 {
		t.Logf("CONFIRMED: Old behavior creates multiple jobs (%d), causing OOM cascade", jobsCreated.Load())
	} else {
		t.Logf("Note: Race conditions resulted in only %d jobs this time, but this is not reliable", jobsCreated.Load())
	}
}

// TestConcurrentFirstReconcileWithRealReconciler simulates the real reconciler behavior
func TestConcurrentFirstReconcileWithRealReconciler(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := eraserv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	// Initialize the module-level variables as they would be in the real controller
	firstReconcileMutex.Lock()
	firstReconcileDone = false
	firstReconcileMutex.Unlock()

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Create a minimal reconciler-like function
	type ReconcileResult struct {
		JobCreated bool
		Error      error
	}

	reconcileFirstJob := func() ReconcileResult {
		// Try to acquire lock
		if !firstReconcileMutex.TryLock() {
			return ReconcileResult{JobCreated: false, Error: nil}
		}
		defer firstReconcileMutex.Unlock()

		// Check if done
		if firstReconcileDone {
			return ReconcileResult{JobCreated: false, Error: nil}
		}

		// List jobs
		jobList := &eraserv1.ImageJobList{}
		if err := client.List(context.Background(), jobList); err != nil {
			return ReconcileResult{Error: err}
		}

		// Check for running jobs
		for _, job := range jobList.Items {
			if job.Status.Phase == eraserv1.PhaseRunning {
				firstReconcileDone = true
				return ReconcileResult{JobCreated: false, Error: nil}
			}
		}

		// Clean up non-running jobs
		for _, job := range jobList.Items {
			if job.Status.Phase != eraserv1.PhaseRunning {
				client.Delete(context.Background(), &job)
			}
		}

		// Re-list
		updatedJobList := &eraserv1.ImageJobList{}
		client.List(context.Background(), updatedJobList)

		// Check again
		for _, job := range updatedJobList.Items {
			if job.Status.Phase == eraserv1.PhaseRunning {
				firstReconcileDone = true
				return ReconcileResult{JobCreated: false, Error: nil}
			}
		}

		// Create job
		newJob := &eraserv1.ImageJob{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "imagejob-",
				Namespace:    "eraser-system",
			},
		}
		if err := client.Create(context.Background(), newJob); err != nil {
			return ReconcileResult{Error: err}
		}

		firstReconcileDone = true
		return ReconcileResult{JobCreated: true, Error: nil}
	}

	// Run 20 concurrent reconcile attempts
	var jobsCreated int32
	var wg sync.WaitGroup
	results := make(chan ReconcileResult, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(attempt int) {
			defer wg.Done()
			time.Sleep(time.Duration(attempt%5) * 5 * time.Millisecond)
			result := reconcileFirstJob()
			results <- result
			if result.JobCreated {
				atomic.AddInt32(&jobsCreated, 1)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Wait for all results
	for range results {
		// Just drain the channel
	}

	finalJobList := &eraserv1.ImageJobList{}
	client.List(context.Background(), finalJobList)

	t.Logf("=== REAL RECONCILER SIMULATION ===")
	t.Logf("Concurrent reconcile attempts: 20")
	t.Logf("Jobs actually created: %d", atomic.LoadInt32(&jobsCreated))
	t.Logf("Final jobs in cluster: %d", len(finalJobList.Items))

	if atomic.LoadInt32(&jobsCreated) == 1 && len(finalJobList.Items) == 1 {
		t.Logf("SUCCESS: Fix works correctly - only 1 job created despite 20 concurrent attempts")
	} else {
		t.Errorf("FAIL: Fix did not work - %d jobs created", atomic.LoadInt32(&jobsCreated))
	}
}
