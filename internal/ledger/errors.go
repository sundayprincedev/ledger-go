package ledger

import "errors"

var (
	ErrEmptyEntryID     = errors.New("entry id is empty")
	ErrTooFewLines      = errors.New("entry must have at least two lines")
	ErrEmptyAccountID   = errors.New("line account id is empty")
	ErrInvalidAmount    = errors.New("line amount must be positive")
	ErrUnknownCurrency  = errors.New("unknown currency code")
	ErrUnknownDirection = errors.New("unknown line direction")
	ErrUnbalancedEntry  = errors.New("entry does not balance")
	ErrAmountOverflow   = errors.New("amount sum overflows int64")
)
