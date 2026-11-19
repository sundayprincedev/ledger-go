package jobs

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func TestCheckReadContextPassesWhileTheContextIsLive(t *testing.T) {
	if err := checkReadContext(context.Background(), 0); err != nil {
		t.Fatalf("a live context must pass, got %v", err)
	}
}

func TestCheckReadContextWrapsTheCancellationWithTheRecordIndex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := checkReadContext(ctx, 3)

	var readErr *apperrors.ReadPipelineError
	if !errors.As(err, &readErr) {
		t.Fatalf("expected a ReadPipelineError, got %T", err)
	}
	if readErr.Step != apperrors.ReadStepContext || readErr.Index != 3 {
		t.Fatalf("unexpected step or index: %+v", readErr)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatal("the cause must still be context.Canceled")
	}
}
