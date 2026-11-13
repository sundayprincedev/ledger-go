package jobs

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

type WorkFunc func(ctx context.Context, job *Job) error

type WorkerPool struct {
	Wg     sync.WaitGroup
	Ctx    context.Context
	Cancel context.CancelFunc
}

func NewWorkerPool(parent context.Context) *WorkerPool {
	ctx, cancel := context.WithCancel(parent)
	return &WorkerPool{Ctx: ctx, Cancel: cancel}
}

func (wp *WorkerPool) Start(workerCount int, jobCh <-chan *Job, errCh chan<- error, resultsCh chan<- JobResult, work WorkFunc) {
	wp.Wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wp.Wg.Done()
			for job := range jobCh {
				wp.runJob(job, errCh, resultsCh, work)
			}
		}()
	}

	go func() {
		wp.Wg.Wait()
		wp.Cancel()
	}()
}

func (wp *WorkerPool) runJob(job *Job, errCh chan<- error, resultsCh chan<- JobResult, work WorkFunc) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			errCh <- apperrors.NewWorkerJobErrorWithStack(job.ID, "panic", fmt.Errorf("%v", r), string(buf[:n]))
		}
	}()

	if err := wp.Ctx.Err(); err != nil {
		errCh <- apperrors.NewWorkerJobError(job.ID, "pool shutting down", err)
		return
	}

	from := JobState(job.State.Load())
	if !job.Transition(StateRunning) {
		errCh <- apperrors.NewTransitionError(job.ID, from.String(), StateRunning.String(), "unexpected state", nil)
		return
	}

	jobCtx, cancel := context.WithTimeout(wp.Ctx, job.Timeout)
	defer cancel()

	err := work(jobCtx, job)
	if err != nil {
		from = JobState(job.State.Load())
		if !job.Transition(StateFailed) {
			errCh <- apperrors.NewTransitionError(job.ID, from.String(), StateFailed.String(), "unexpected state", nil)
		}
		errCh <- apperrors.NewWorkerJobError(job.ID, "job failed", err)
		resultsCh <- JobResult{JobID: job.ID, Success: false, Err: err}
		return
	}

	from = JobState(job.State.Load())
	if !job.Transition(StateCompleted) {
		errCh <- apperrors.NewTransitionError(job.ID, from.String(), StateCompleted.String(), "unexpected state", nil)
		return
	}

	resultsCh <- JobResult{JobID: job.ID, Success: true}
}
