package apperrors

import "errors"

var (
	ErrEmptyEntryID      = errors.New("entry id is empty")
	ErrTooFewLines       = errors.New("entry must have at least two lines")
	ErrEmptyAccountID    = errors.New("line account id is empty")
	ErrLineInvalidAmount = errors.New("line amount must be positive")
	ErrUnknownCurrency   = errors.New("unknown currency code")
	ErrUnknownDirection  = errors.New("unknown line direction")
	ErrUnbalancedEntry   = errors.New("entry does not balance")
	ErrAmountOverflow    = errors.New("amount sum overflows int64")

	ErrJobNotFound              = errors.New("job not found")
	ErrJobStateCorrupted        = errors.New("job state is not a recognized type")
	ErrJobNotPending            = errors.New("job is not pending")
	ErrJobConcurrentStateChange = errors.New("job state changed concurrently")
)

type JobError struct {
	JobID string
	Err   error
}

func (e *JobError) Error() string {
	return "job " + e.JobID + ": " + e.Err.Error()
}

func (e *JobError) Unwrap() error {
	return e.Err
}

type JobNotPendingError struct {
	JobID string
	State string
}

func (e *JobNotPendingError) Error() string {
	return "job " + e.JobID + ": state is " + e.State + ", want pending"
}

func (e *JobNotPendingError) Unwrap() error {
	return ErrJobNotPending
}
