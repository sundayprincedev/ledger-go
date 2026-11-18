package apperrors

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidTransition = errors.New("invalid job state transition")
	ErrStaleJob          = errors.New("job made no progress; owning worker presumed lost")

	ErrJobAlreadyInFlight  = errors.New("job is already in flight")
	ErrJobAlreadyCompleted = errors.New("job already completed")
	ErrInvalidPayload      = errors.New("job payload is not the expected type")

	ErrUnknownOpType        = errors.New("unknown operation type")
	ErrOpTypeNotSupported   = errors.New("operation type is not supported yet")
	ErrIntentInvalidAmount  = errors.New("intent amount must be positive")
	ErrIntentMissingAccount = errors.New("intent is missing a required account")
	ErrIntentSameAccount    = errors.New("intent moves money to the same account")
	ErrIntentNoRecipients   = errors.New("bulk intent has no recipients")
	ErrIntentInvalidWeight  = errors.New("recipient weight must be positive")
)

type WorkerJobError struct {
	JobID     string
	Reason    string
	Err       error
	Stack     string
	Timestamp time.Time
}

func (e *WorkerJobError) Error() string {
	var b strings.Builder
	b.WriteString("job ")
	b.WriteString(e.JobID)
	b.WriteString(": ")
	b.WriteString(e.Reason)
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}
	return b.String()
}

func (e *WorkerJobError) Unwrap() error {
	return e.Err
}

func NewWorkerJobError(jobID, reason string, err error) *WorkerJobError {
	return &WorkerJobError{
		JobID:     jobID,
		Reason:    reason,
		Err:       err,
		Timestamp: time.Now(),
	}
}

func NewWorkerJobErrorWithStack(jobID, reason string, err error, stack string) *WorkerJobError {
	return &WorkerJobError{
		JobID:     jobID,
		Reason:    reason,
		Err:       err,
		Stack:     stack,
		Timestamp: time.Now(),
	}
}

type PipelineStep string

const (
	PipelineStepRegister      PipelineStep = "register"
	PipelineStepIntent        PipelineStep = "intent"
	PipelineStepCurrency      PipelineStep = "currency"
	PipelineStepKYC           PipelineStep = "kyc"
	PipelineStepDispatch      PipelineStep = "dispatch"
	PipelineStepSplit         PipelineStep = "split"
	PipelineStepBuildEntry    PipelineStep = "build_entry"
	PipelineStepValidateEntry PipelineStep = "validate_entry"
	PipelineStepMarshal       PipelineStep = "marshal"
	PipelineStepAppend        PipelineStep = "append"
	PipelineStepComplete      PipelineStep = "complete"
)

type IntentStep string

const (
	IntentStepType       IntentStep = "type"
	IntentStepAmount     IntentStep = "amount"
	IntentStepCurrency   IntentStep = "currency"
	IntentStepAccounts   IntentStep = "accounts"
	IntentStepRecipients IntentStep = "recipients"
)

type IntentValidationError struct {
	Step           IntentStep
	RecipientIndex int
	Err            error
}

func (e *IntentValidationError) Error() string {
	if e.RecipientIndex < 0 {
		return fmt.Sprintf("intent validation failed at %s: %v", e.Step, e.Err)
	}
	return fmt.Sprintf("intent validation failed at %s (recipient %d): %v", e.Step, e.RecipientIndex, e.Err)
}

func (e *IntentValidationError) Unwrap() error {
	return e.Err
}

type OpTypeError struct {
	Type string
}

func (e *OpTypeError) Error() string {
	var b strings.Builder
	b.WriteString("unknown operation type: ")
	b.WriteString(e.Type)
	return b.String()
}

func (e *OpTypeError) Unwrap() error {
	return ErrUnknownOpType
}

type UnsupportedOpTypeError struct {
	Type string
}

func (e *UnsupportedOpTypeError) Error() string {
	var b strings.Builder
	b.WriteString(ErrOpTypeNotSupported.Error())
	b.WriteString(": ")
	b.WriteString(e.Type)
	return b.String()
}

func (e *UnsupportedOpTypeError) Unwrap() error {
	return ErrOpTypeNotSupported
}

type JobPipelineError struct {
	JobID string
	Step  PipelineStep
	Err   error
}

func (e *JobPipelineError) Error() string {
	var b strings.Builder
	b.WriteString("job ")
	b.WriteString(e.JobID)
	b.WriteString(": pipeline failed at ")
	b.WriteString(string(e.Step))
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	return b.String()
}

func (e *JobPipelineError) Unwrap() error {
	return e.Err
}

type TransitionError struct {
	JobID     string
	From      string
	To        string
	Reason    string
	Err       error
	Timestamp time.Time
}

func (e *TransitionError) Error() string {
	var b strings.Builder
	b.WriteString("job ")
	b.WriteString(e.JobID)
	b.WriteString(": transition ")
	b.WriteString(e.From)
	b.WriteString(" -> ")
	b.WriteString(e.To)
	b.WriteString(": ")
	b.WriteString(e.Reason)
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}
	return b.String()
}

func (e *TransitionError) Unwrap() error {
	if e.Err != nil {
		return e.Err
	}
	return ErrInvalidTransition
}

func NewTransitionError(jobID, from, to, reason string, err error) *TransitionError {
	return &TransitionError{
		JobID:     jobID,
		From:      from,
		To:        to,
		Reason:    reason,
		Err:       err,
		Timestamp: time.Now(),
	}
}
