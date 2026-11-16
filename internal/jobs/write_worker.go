package jobs

import (
	"context"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

type WriteWorker struct {
	*WorkerPool
}

func NewWriteWorker(parent context.Context) *WriteWorker {
	return &WriteWorker{WorkerPool: NewWorkerPool(parent)}
}

func (wr *WriteWorker) WriteWork(registry *ledger.IDRegistry, store *ledger.AccountStore) WorkFunc {
	return func(ctx context.Context, job *Job) error {
		intent, ok := job.Payload.(Intent)
		if !ok {
			return apperrors.NewWorkerJobError(job.ID, "invalid payload", apperrors.ErrInvalidPayload)
		}
		return wr.StartWritePipeline(ctx, registry, store, job.ID, intent)
	}
}

func (wr *WriteWorker) StartWritePipeline(ctx context.Context, IDRegistry *ledger.IDRegistry, AccountRegistry *ledger.AccountStore, jobID string, intent Intent) error {
	var claimed bool

	defer func() {
		if claimed {
			// TODO: once the journal append exists, release only if the entry was not committed
			_ = IDRegistry.Release(jobID)
		}
	}()

	pipelineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := intent.Validate(); err != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepIntent, Err: err}
	}

	if err := checkContext(pipelineCtx, jobID, apperrors.PipelineStepRegister); err != nil {
		return err
	}

	RegisterResult, RegisterErr := IDRegistry.Register(jobID)
	if RegisterErr != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepRegister, Err: RegisterErr}
	}

	if !RegisterResult.New {
		switch RegisterResult.State {
		case ledger.StateCompleted:
			// TODO: build the results pipeline this collects the original result from, so a completed duplicate returns Ack and the rest of the flow to the user instead of an error
			return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepRegister, Err: apperrors.ErrJobAlreadyCompleted}
		case ledger.StateInFlight:
			return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepRegister, Err: apperrors.ErrJobAlreadyInFlight}
		default:
			return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepRegister, Err: apperrors.ErrJobStateCorrupted}
		}
	}

	claimed = true

	if err := kycChecker(AccountRegistry, intent); err != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepKYC, Err: err}
	}

	if intent.Type == OpBulkTransfer {

	}

	

	if err := IDRegistry.Complete(jobID); err != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepComplete, Err: err}
	}
	claimed = false

	return nil
}
