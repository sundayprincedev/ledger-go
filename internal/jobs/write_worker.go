package jobs

import (
	"context"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/ledger"
	"github.com/sundayprincedev/ledger/internal/money"
)

type WriteWorker struct {
	*WorkerPool
}

func NewWriteWorker(parent context.Context) *WriteWorker {
	return &WriteWorker{WorkerPool: NewWorkerPool(parent)}
}

func (wr *WriteWorker) WriteWork(registry *ledger.IDRegistry, store *ledger.AccountStore, writer ledger.Journal) WorkFunc {
	return func(ctx context.Context, job *Job) error {
		intent, ok := job.Payload.(Intent)
		if !ok {
			return apperrors.NewWorkerJobError(job.ID, "invalid payload", apperrors.ErrInvalidPayload)
		}
		return wr.StartWritePipeline(ctx, registry, store, writer, job.ID, intent)
	}
}

func (wr *WriteWorker) StartWritePipeline(ctx context.Context, IDRegistry *ledger.IDRegistry, AccountRegistry *ledger.AccountStore, Writer ledger.Journal, jobID string, intent Intent) error {
	var claimed bool

	defer func() {
		if claimed {
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

	if intent.Type == OpBulkTransfer {
		// TODO: bulk path: its own currency check, Allocate, KYC on each share, then build the lines
	}

	if intent.Type != OpSingleTransfer {
		return &apperrors.JobPipelineError{
			JobID: jobID,
			Step:  apperrors.PipelineStepDispatch,
			Err:   &apperrors.UnsupportedOpTypeError{Type: string(intent.Type)},
		}
	}

	if currCheckErrorIntent := AccountRegistry.CheckCurrency(intent.FromAccountID, intent.CurrencyCode); currCheckErrorIntent != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepCurrency, Err: currCheckErrorIntent}
	}

	if currCheckErrorReceiver := AccountRegistry.CheckCurrency(intent.ToAccountID, intent.CurrencyCode); currCheckErrorReceiver != nil {
		//TODO: Run currency conversion pipeline so currencies match (with roundrule)
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepCurrency, Err: currCheckErrorReceiver}
	}

	if err := kycChecker(AccountRegistry, intent); err != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepKYC, Err: err}
	}

	splitCut, splitErr := money.SplitHouseCut(intent.Amount)
	if splitErr != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepSplit, Err: splitErr}
	}

	if len(splitCut) != 2 {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepSplit, Err: apperrors.ErrSplitShareCount}
	}

	if err := checkContext(pipelineCtx, jobID, apperrors.PipelineStepBuildEntry); err != nil {
		return err
	}

	mainEntry, buildErr := buildDirectEntry(AccountRegistry, jobID, intent, splitCut)
	if buildErr != nil {
		return buildErr
	}

	if entryValidationErr := mainEntry.Validate(); entryValidationErr != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepValidateEntry, Err: entryValidationErr}
	}

	if err := checkContext(pipelineCtx, jobID, apperrors.PipelineStepMarshal); err != nil {
		return err
	}

	payload, marshalErr := mainEntry.Marshal()
	if marshalErr != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepMarshal, Err: marshalErr}
	}

	if err := checkContext(pipelineCtx, jobID, apperrors.PipelineStepAppend); err != nil {
		return err
	}

	if appendError := Writer.Append(payload); appendError != nil {
		recoveryRequired, outcomeUnknown := classifyAppendError(appendError)
		if outcomeUnknown {
			claimed = false
		}
		return &apperrors.JobPipelineError{
			JobID: jobID,
			Step:  apperrors.PipelineStepAppend,
			Err: &apperrors.AppendFailureError{
				Err:              appendError,
				RecoveryRequired: recoveryRequired,
				OutcomeUnknown:   outcomeUnknown,
			},
		}
	}

	claimed = false

	if err := IDRegistry.Complete(jobID); err != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepComplete, Err: err}
	}

	return nil
}
