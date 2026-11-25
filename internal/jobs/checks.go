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

func checkReadContext(ctx context.Context, index int) error {
	if err := ctx.Err(); err != nil {
		return &apperrors.ReadPipelineError{Step: apperrors.ReadStepContext, Index: index, Err: err}
	}
	return nil
}

func checkResolveContext(ctx context.Context, index int) error {
	if err := ctx.Err(); err != nil {
		return &apperrors.ResolvePipelineError{Step: apperrors.ResolveStepContext, Index: index, Err: err}
	}
	return nil
}
