package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrWriterPoisoned  = errors.New("log writer is poisoned due to previous write/sync failure; recovery required")
	ErrCorruptRecord   = errors.New("corrupted record encountered in middle of file")
	ErrPayloadTooLarge = errors.New("payload length exceeds maximum allowed size")
)

type WritePipelineStep string

const (
	StepOpenFile       WritePipelineStep = "open_file"
	StepOpenDir        WritePipelineStep = "open_dir"
	StepSyncDir        WritePipelineStep = "sync_dir"
	StepCloseDirHandle WritePipelineStep = "close_dir_handle"
	StepWriteRecord    WritePipelineStep = "write_record"
	StepSyncRecord     WritePipelineStep = "sync_record"
)

type WritePipelineError struct {
	Step WritePipelineStep
	Err  error
}

func (e *WritePipelineError) Error() string {
	return fmt.Sprintf("write pipeline failed at %s: %v", e.Step, e.Err)
}

func (e *WritePipelineError) Unwrap() error {
	return e.Err
}

type RecoveryStep string

const (
	RecoverStepOpenFile     RecoveryStep = "open_file"
	RecoverStepStat         RecoveryStep = "stat_file"
	RecoverStepReadHeader   RecoveryStep = "read_header"
	RecoverStepCheckTail    RecoveryStep = "check_tail"
	RecoverStepHeaderCRC    RecoveryStep = "header_crc"
	RecoverStepPayloadSize  RecoveryStep = "payload_size"
	RecoverStepReadPayload  RecoveryStep = "read_payload"
	RecoverStepPayloadCRC   RecoveryStep = "payload_crc"
	RecoverStepTruncate     RecoveryStep = "truncate"
	RecoverStepSyncTruncate RecoveryStep = "sync_truncate"
)

type RecoveryPipelineError struct {
	Step   RecoveryStep
	Offset int64
	Err    error
}

func (e *RecoveryPipelineError) Error() string {
	if e.Offset < 0 {
		return fmt.Sprintf("journal recovery failed at %s: %v", e.Step, e.Err)
	}
	return fmt.Sprintf("journal recovery failed at %s (offset %d): %v", e.Step, e.Offset, e.Err)
}

func (e *RecoveryPipelineError) Unwrap() error {
	return e.Err
}
