package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrAccountInvalidAmount = errors.New("invalid amount")
	ErrAccountLocked        = errors.New("account locked")
	ErrAccountNotFound      = errors.New("account not found")
)

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
