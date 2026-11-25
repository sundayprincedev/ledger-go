package apperrors

import (
	"errors"
	"testing"
)

func TestResolvePipelineErrorMessageNamesTheStepRecordAndEntry(t *testing.T) {
	err := &ResolvePipelineError{Step: ResolveStepApply, Index: 4, EntryID: "job-9", Err: ErrBalanceOverflow}

	want := "pending resolution failed at apply (record 4, entry job-9): balance overflows int64"
	if err.Error() != want {
		t.Fatalf("want %q, got %q", want, err.Error())
	}
	if !errors.Is(err, ErrBalanceOverflow) {
		t.Fatal("the cause must stay reachable")
	}
}

func TestResolvePipelineErrorWithoutARecordStillNamesTheEntry(t *testing.T) {
	err := &ResolvePipelineError{Step: ResolveStepFail, Index: -1, EntryID: "job-2", Err: ErrJobNotPending}

	want := "pending resolution failed at fail (entry job-2): job is not pending"
	if err.Error() != want {
		t.Fatalf("want %q, got %q", want, err.Error())
	}
}

func TestResolvePipelineErrorWithNeitherRecordNorEntry(t *testing.T) {
	err := &ResolvePipelineError{Step: ResolveStepRecover, Index: -1, Err: ErrCorruptRecord}

	want := "pending resolution failed at recover: corrupted record encountered in middle of file"
	if err.Error() != want {
		t.Fatalf("want %q, got %q", want, err.Error())
	}
}

func TestUnexpectedJobStateErrorUnwrapsToItsSentinel(t *testing.T) {
	err := &UnexpectedJobStateError{JobID: "job-1", State: "in_flight"}

	if !errors.Is(err, ErrUnexpectedJobState) {
		t.Fatal("expected ErrUnexpectedJobState")
	}
	want := "job job-1: state is in_flight during resolution, want pending or completed"
	if err.Error() != want {
		t.Fatalf("want %q, got %q", want, err.Error())
	}
}
