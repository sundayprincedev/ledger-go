package apperrors

import (
	"errors"
	"strconv"
	"strings"
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
	var b strings.Builder
	b.WriteString("insufficient balance on ")
	b.WriteString(e.AccountID)
	b.WriteString(": short by ")
	b.WriteString(strconv.FormatInt(e.Deficit, 10))
	b.WriteString(" units")
	return b.String()
}

type AccountLockedError struct {
	AccountID string
	Reason    string
}

func (e *AccountLockedError) Error() string {
	var b strings.Builder
	b.WriteString("account ")
	b.WriteString(e.AccountID)
	b.WriteString(" locked: ")
	b.WriteString(e.Reason)
	return b.String()
}
