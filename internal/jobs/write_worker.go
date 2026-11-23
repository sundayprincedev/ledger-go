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

func (wr *WriteWorker) WriteWork(registry *ledger.IDRegistry, store *ledger.AccountStore, balances *ledger.Balances, writer ledger.Journal) WorkFunc {
	return func(ctx context.Context, job *Job) error {
		intent, ok := job.Payload.(Intent)
		if !ok {
			return apperrors.NewWorkerJobError(job.ID, "invalid payload", apperrors.ErrInvalidPayload)
		}
		ack, err := wr.StartWritePipeline(ctx, registry, store, balances, writer, job.ID, intent)
		if err != nil {
			return err
		}
		job.Ack = &ack
		return nil
	}
}

func (wr *WriteWorker) StartWritePipeline(ctx context.Context, IDRegistry *ledger.IDRegistry, AccountRegistry *ledger.AccountStore, Balances *ledger.Balances, Writer ledger.Journal, jobID string, intent Intent) (Ack, error) {
	var claimed bool

	defer func() {
		if claimed {
			_ = IDRegistry.Release(jobID)
		}
	}()

	pipelineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := intent.Validate(); err != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepIntent, Err: err}
	}

	if err := checkContext(pipelineCtx, jobID, apperrors.PipelineStepRegister); err != nil {
		return Ack{}, err
	}

	RegisterResult, RegisterErr := IDRegistry.Register(jobID)
	if RegisterErr != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepRegister, Err: RegisterErr}
	}

	if !RegisterResult.New {
		switch RegisterResult.State {
		case ledger.StateCompleted:
			return newAck(jobID, AckAlreadyCompleted), nil
		case ledger.StatePending:
			return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepRegister, Err: apperrors.ErrJobOutcomePending}
		case ledger.StateInFlight:
			return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepRegister, Err: apperrors.ErrJobAlreadyInFlight}
		default:
			return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepRegister, Err: apperrors.ErrJobStateCorrupted}
		}
	}

	claimed = true

	if intent.Type == OpBulkTransfer {
		// TODO: bulk path: its own currency check, Allocate, KYC on each share, then build the lines
	}

	if intent.Type != OpSingleTransfer {
		return Ack{}, &apperrors.JobPipelineError{
			JobID: jobID,
			Step:  apperrors.PipelineStepDispatch,
			Err:   &apperrors.UnsupportedOpTypeError{Type: string(intent.Type)},
		}
	}

	if currCheckErrorIntent := AccountRegistry.CheckCurrency(intent.FromAccountID, intent.CurrencyCode); currCheckErrorIntent != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepCurrency, Err: currCheckErrorIntent}
	}

	if currCheckErrorReceiver := AccountRegistry.CheckCurrency(intent.ToAccountID, intent.CurrencyCode); currCheckErrorReceiver != nil {
		//TODO: Run currency conversion pipeline so currencies match (with roundrule)
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepCurrency, Err: currCheckErrorReceiver}
	}

	if err := kycChecker(AccountRegistry, intent); err != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepKYC, Err: err}
	}

	splitCut, splitErr := money.SplitHouseCut(intent.Amount)
	if splitErr != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepSplit, Err: splitErr}
	}

	if len(splitCut) != 2 {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepSplit, Err: apperrors.ErrSplitShareCount}
	}

	if err := checkContext(pipelineCtx, jobID, apperrors.PipelineStepBuildEntry); err != nil {
		return Ack{}, err
	}

	mainEntry, buildErr := buildDirectEntry(AccountRegistry, jobID, intent, splitCut)
	if buildErr != nil {
		return Ack{}, buildErr
	}

	if entryValidationErr := mainEntry.Validate(); entryValidationErr != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepValidateEntry, Err: entryValidationErr}
	}

	if err := checkContext(pipelineCtx, jobID, apperrors.PipelineStepMarshal); err != nil {
		return Ack{}, err
	}

	payload, marshalErr := mainEntry.Marshal()
	if marshalErr != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepMarshal, Err: marshalErr}
	}

	if err := checkContext(pipelineCtx, jobID, apperrors.PipelineStepAppend); err != nil {
		return Ack{}, err
	}

	if appendError := Writer.Append(payload); appendError != nil {
		recoveryRequired, outcomeUnknown := classifyAppendError(appendError)
		if outcomeUnknown {
			claimed = false
			_ = IDRegistry.MarkPending(jobID)
		}
		return Ack{}, &apperrors.JobPipelineError{
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

	if err := Balances.Apply(mainEntry); err != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepApply, Err: err}
	}

	if err := IDRegistry.Complete(jobID); err != nil {
		return Ack{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepComplete, Err: err}
	}

	return newAck(jobID, AckCompleted), nil
}
