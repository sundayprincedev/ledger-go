package jobs

import (
	"errors"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func classifyAppendError(err error) (recoveryRequired, outcomeUnknown bool) {
	if errors.Is(err, apperrors.ErrWriterPoisoned) {
		return true, false
	}

	var writeErr *apperrors.WritePipelineError
	if errors.As(err, &writeErr) {
		switch writeErr.Step {
		case apperrors.StepWriteRecord:
			return true, false
		case apperrors.StepSyncRecord:
			return true, true
		}
	}

	return false, false
}
