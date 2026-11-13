package apperrors

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidTransition = errors.New("invalid job state transition")
	ErrStaleJob          = errors.New("job made no progress; owning worker presumed lost")
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
