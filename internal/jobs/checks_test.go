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

func TestCheckResolveContextPassesWhileTheContextIsLive(t *testing.T) {
	if err := checkResolveContext(context.Background(), 0); err != nil {
		t.Fatalf("a live context must pass, got %v", err)
	}
}

func TestCheckResolveContextWrapsTheCancellationWithTheRecordIndex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := checkResolveContext(ctx, 2)

	var resolveErr *apperrors.ResolvePipelineError
	if !errors.As(err, &resolveErr) {
		t.Fatalf("expected a ResolvePipelineError, got %T", err)
	}
	if resolveErr.Step != apperrors.ResolveStepContext || resolveErr.Index != 2 {
		t.Fatalf("unexpected step or index: %+v", resolveErr)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatal("the cause must still be context.Canceled")
	}
}
