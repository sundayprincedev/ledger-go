package jobs

import (
	"context"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func checkContext(ctx context.Context, jobID string, step apperrors.PipelineStep) error {
	if err := ctx.Err(); err != nil {
		return &apperrors.JobPipelineError{JobID: jobID, Step: step, Err: err}
	}
	return nil
}
