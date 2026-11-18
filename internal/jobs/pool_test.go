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

func TestRunJobUnknownOutcomeMovesJobToOutcomePending(t *testing.T) {
	wp := NewWorkerPool(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wp.Start(1, jobCh, errCh, resultsCh, func(ctx context.Context, job *Job) error {
		return &apperrors.AppendFailureError{Err: errors.New("fsync failed"), OutcomeUnknown: true}
	})

	job := &Job{ID: "unknown-job"}
	jobCh <- job
	close(jobCh)
	wp.Wg.Wait()

	if state := JobState(job.State.Load()); state != StateOutcomePending {
		t.Fatalf("an unknown outcome must leave the job outcome_pending, got %s", state)
	}

	var workerErr *apperrors.WorkerJobError
	if err := <-errCh; !errors.As(err, &workerErr) || workerErr.Reason != "outcome unknown" {
		t.Fatalf("expected an 'outcome unknown' worker error, got: %v", err)
	}

	if result := <-resultsCh; result.Success || !errors.Is(result.Err, apperrors.ErrOutcomeUnknown) {
		t.Fatalf("the result must not be a success and must keep the marker, got: %+v", result)
	}

	if !job.Transition(StateCommitted) {
		t.Fatal("outcome_pending is not terminal, so recovery must be able to move it on to committed")
	}
}

func TestRunJobPlainFailureStillEndsFailed(t *testing.T) {
	wp := NewWorkerPool(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wp.Start(1, jobCh, errCh, resultsCh, func(ctx context.Context, job *Job) error {
		return errors.New("boom")
	})

	job := &Job{ID: "failed-job"}
	jobCh <- job
	close(jobCh)
	wp.Wg.Wait()

	if state := JobState(job.State.Load()); state != StateFailed {
		t.Fatalf("an ordinary failure must end failed, got %s", state)
	}

	var workerErr *apperrors.WorkerJobError
	if err := <-errCh; !errors.As(err, &workerErr) || workerErr.Reason != "job failed" {
		t.Fatalf("expected a 'job failed' worker error, got: %v", err)
	}

	if job.Transition(StateCommitted) {
		t.Fatal("failed is terminal, so it must not move on")
	}
}
