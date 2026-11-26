package apperrors

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrWriterPoisoned  = errors.New("log writer is poisoned due to previous write/sync failure; recovery required")
	ErrCorruptRecord   = errors.New("corrupted record encountered in middle of file")
	ErrPayloadTooLarge = errors.New("payload length exceeds maximum allowed size")

	ErrRecoveryRequired = errors.New("journal needs recovery before further writes")
	ErrOutcomeUnknown   = errors.New("the entry may or may not be on disk")
)

type AppendFailureError struct {
	Err              error
	RecoveryRequired bool
	OutcomeUnknown   bool

	ResolvedCompleted []string
	ResolvedFailed    []string
	ResolverErr       error
}

func (e *AppendFailureError) Error() string {
	var b strings.Builder
	b.WriteString("append failed")
	if e.RecoveryRequired {
		b.WriteString(", recovery required")
	}
	if e.OutcomeUnknown {
		b.WriteString(", outcome unknown")
	}
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	if e.ResolverErr != nil {
		b.WriteString("; resolving pending jobs also failed: ")
		b.WriteString(e.ResolverErr.Error())
	}
	return b.String()
}

func (e *AppendFailureError) Unwrap() error {
	return e.Err
}

func (e *AppendFailureError) Is(target error) bool {
	switch target {
	case ErrRecoveryRequired:
		return e.RecoveryRequired
	case ErrOutcomeUnknown:
		return e.OutcomeUnknown
	}
	return false
}

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

type ReadPipelineStep string

const (
	ReadStepRecover  ReadPipelineStep = "recover"
	ReadStepContext  ReadPipelineStep = "context"
	ReadStepDecode   ReadPipelineStep = "decode"
	ReadStepRegister ReadPipelineStep = "register"
	ReadStepApply    ReadPipelineStep = "apply"
)

type ReadPipelineError struct {
	Step    ReadPipelineStep
	Index   int
	EntryID string
	Err     error
}

func (e *ReadPipelineError) Error() string {
	var b strings.Builder
	b.WriteString("startup read failed at ")
	b.WriteString(string(e.Step))
	if e.Index >= 0 {
		b.WriteString(fmt.Sprintf(" (record %d", e.Index))
		if e.EntryID != "" {
			b.WriteString(", entry ")
			b.WriteString(e.EntryID)
		}
		b.WriteString(")")
	}
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	return b.String()
}

func (e *ReadPipelineError) Unwrap() error {
	return e.Err
}

var ErrDuplicateJournalEntry = errors.New("entry appears more than once in the journal")

type DuplicateJournalEntryError struct {
	EntryID string
	Index   int
}

func (e *DuplicateJournalEntryError) Error() string {
	return fmt.Sprintf("entry %s at record %d is a duplicate of an earlier record", e.EntryID, e.Index)
}

func (e *DuplicateJournalEntryError) Unwrap() error {
	return ErrDuplicateJournalEntry
}

type ResolveStep string

const (
	ResolveStepRecover  ResolveStep = "recover"
	ResolveStepContext  ResolveStep = "context"
	ResolveStepDecode   ResolveStep = "decode"
	ResolveStepComplete ResolveStep = "complete"
	ResolveStepApply    ResolveStep = "apply"
	ResolveStepFail     ResolveStep = "fail"
)

type ResolvePipelineError struct {
	Step    ResolveStep
	Index   int
	EntryID string
	Err     error
}

func (e *ResolvePipelineError) Error() string {
	var b strings.Builder
	b.WriteString("pending resolution failed at ")
	b.WriteString(string(e.Step))
	if e.Index >= 0 {
		b.WriteString(fmt.Sprintf(" (record %d", e.Index))
		if e.EntryID != "" {
			b.WriteString(", entry ")
			b.WriteString(e.EntryID)
		}
		b.WriteString(")")
	} else if e.EntryID != "" {
		b.WriteString(" (entry ")
		b.WriteString(e.EntryID)
		b.WriteString(")")
	}
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	return b.String()
}

func (e *ResolvePipelineError) Unwrap() error {
	return e.Err
}

var ErrUnexpectedJobState = errors.New("job is in a state that cannot occur during resolution")

type UnexpectedJobStateError struct {
	JobID string
	State string
}

func (e *UnexpectedJobStateError) Error() string {
	return "job " + e.JobID + ": state is " + e.State + " during resolution, want pending or completed"
}

func (e *UnexpectedJobStateError) Unwrap() error {
	return ErrUnexpectedJobState
}
