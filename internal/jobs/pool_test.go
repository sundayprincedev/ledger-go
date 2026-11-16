package jobs

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func TestRunJobPanicFailsJobAndReportsResult(t *testing.T) {
	wp := NewWorkerPool(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wp.Start(1, jobCh, errCh, resultsCh, func(ctx context.Context, job *Job) error {
		panic("boom")
	})

	job := &Job{ID: "panic-job"}
	jobCh <- job
	close(jobCh)
	wp.Wg.Wait()

	var workerErr *apperrors.WorkerJobError
	if err := <-errCh; !errors.As(err, &workerErr) || workerErr.Reason != "panic" || workerErr.Stack == "" {
		t.Fatalf("expected a panic WorkerJobError carrying a stack, got: %v", err)
	}

	result := <-resultsCh
	if result.Success || result.JobID != "panic-job" {
		t.Fatalf("expected a failed result for panic-job, got: %+v", result)
	}

	if state := JobState(job.State.Load()); state != StateFailed {
		t.Fatalf("panicked job should end in failed, got %s", state)
	}
}
