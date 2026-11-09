package accounts

import (
	"errors"
	"fmt"
)

var ErrInvalidAmount = errors.New("invalid amount")
var ErrAccountLocked = errors.New("account locked")
var ErrAccountNotFound = errors.New("account not found")

type InsufficientFundsError struct {
	AccountID       string
	AmountRequested int64
	Deficit         int64
}

func (e *InsufficientFundsError) Error() string {
	return fmt.Sprintf("insufficient balance on %s: short by %d units", e.AccountID, e.Deficit)
}

type AccountLockedError struct {
	AccountID string
	Reason    string
}

func (e *AccountLockedError) Error() string {
	return fmt.Sprintf("account %s locked: %s", e.AccountID, e.Reason)
}
