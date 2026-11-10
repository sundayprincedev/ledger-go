package apperrors

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
	ErrJobNotInFlight           = errors.New("job is not in flight")
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

type JobNotInFlightError struct {
	JobID string
	State string
}

func (e *JobNotInFlightError) Error() string {
	return "job " + e.JobID + ": state is " + e.State + ", want in_flight"
}

func (e *JobNotInFlightError) Unwrap() error {
	return ErrJobNotInFlight
}

type ValidationStep string

const (
	StepEntryID   ValidationStep = "entry_id"
	StepLineCount ValidationStep = "line_count"
	StepAccountID ValidationStep = "account_id"
	StepAmount    ValidationStep = "amount"
	StepCurrency  ValidationStep = "currency"
	StepDirection ValidationStep = "direction"
	StepOverflow  ValidationStep = "overflow"
	StepBalance   ValidationStep = "balance"
)

type EntryValidationError struct {
	Step      ValidationStep
	LineIndex int
	Err       error
}

func (e *EntryValidationError) Error() string {
	if e.LineIndex < 0 {
		return fmt.Sprintf("entry validation failed at %s: %v", e.Step, e.Err)
	}
	return fmt.Sprintf("entry validation failed at %s (line %d): %v", e.Step, e.LineIndex, e.Err)
}

func (e *EntryValidationError) Unwrap() error {
	return e.Err
}

type EntryDecodeError struct {
	Err error
}

func (e *EntryDecodeError) Error() string {
	var b strings.Builder
	b.WriteString("decode entry: ")
	b.WriteString(e.Err.Error())
	return b.String()
}

func (e *EntryDecodeError) Unwrap() error {
	return e.Err
}

type CurrencyError struct {
	Code string
}

func (e *CurrencyError) Error() string {
	var b strings.Builder
	b.WriteString("unknown currency code: ")
	b.WriteString(e.Code)
	return b.String()
}

func (e *CurrencyError) Unwrap() error {
	return ErrUnknownCurrency
}

type DirectionError struct {
	Direction string
}

func (e *DirectionError) Error() string {
	var b strings.Builder
	b.WriteString("unknown line direction: ")
	b.WriteString(e.Direction)
	return b.String()
}

func (e *DirectionError) Unwrap() error {
	return ErrUnknownDirection
}

type BalanceError struct {
	CurrencyCode string
	Debits       int64
	Credits      int64
}

func (e *BalanceError) Error() string {
	var b strings.Builder
	b.WriteString("entry does not balance for ")
	b.WriteString(e.CurrencyCode)
	b.WriteString(": debits=")
	b.WriteString(strconv.FormatInt(e.Debits, 10))
	b.WriteString(" credits=")
	b.WriteString(strconv.FormatInt(e.Credits, 10))
	return b.String()
}

func (e *BalanceError) Unwrap() error {
	return ErrUnbalancedEntry
}
