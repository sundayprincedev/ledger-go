package jobs

import (
	"errors"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func TestClassifyAppendError(t *testing.T) {
	cause := errors.New("io failure")

	tests := []struct {
		name         string
		err          error
		wantRecovery bool
		wantUnknown  bool
	}{
		{"write_record", &apperrors.WritePipelineError{Step: apperrors.StepWriteRecord, Err: cause}, true, false},
		{"sync_record", &apperrors.WritePipelineError{Step: apperrors.StepSyncRecord, Err: cause}, true, true},
		{"writer poisoned", apperrors.ErrWriterPoisoned, true, false},
		{"open_file", &apperrors.WritePipelineError{Step: apperrors.StepOpenFile, Err: cause}, false, false},
		{"open_dir", &apperrors.WritePipelineError{Step: apperrors.StepOpenDir, Err: cause}, false, false},
		{"sync_dir", &apperrors.WritePipelineError{Step: apperrors.StepSyncDir, Err: cause}, false, false},
		{"close_dir_handle", &apperrors.WritePipelineError{Step: apperrors.StepCloseDirHandle, Err: cause}, false, false},
		{"payload too large", apperrors.ErrPayloadTooLarge, false, false},
		{"anything else", cause, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recovery, unknown := classifyAppendError(tc.err)
			if recovery != tc.wantRecovery || unknown != tc.wantUnknown {
				t.Fatalf("want recovery=%v unknown=%v, got recovery=%v unknown=%v", tc.wantRecovery, tc.wantUnknown, recovery, unknown)
			}
		})
	}
}

func TestClassifyAppendErrorSeesThroughWrapping(t *testing.T) {
	inner := &apperrors.WritePipelineError{Step: apperrors.StepSyncRecord, Err: errors.New("fsync failed")}
	wrapped := &apperrors.JobPipelineError{JobID: "job-1", Step: apperrors.PipelineStepAppend, Err: inner}

	recovery, unknown := classifyAppendError(wrapped)
	if !recovery || !unknown {
		t.Fatalf("a sync failure must be recognised through wrapping, got recovery=%v unknown=%v", recovery, unknown)
	}
}
