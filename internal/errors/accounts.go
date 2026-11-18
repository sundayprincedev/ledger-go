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
	ErrAccountExists        = errors.New("account already exists")
	ErrAccountIDRequired    = errors.New("account id is required")
	ErrTierLimitsNotFound   = errors.New("no limits defined for tier")

	ErrTransferLimitExceeded = errors.New("amount exceeds the single transfer limit")
	ErrReceiveLimitExceeded  = errors.New("amount exceeds the single receive limit")

	ErrNoAccountForKind    = errors.New("no account of this kind for the currency")
	ErrSystemAccountExists = errors.New("a system account of this kind already exists for the currency")

	ErrCurrencyMismatch = errors.New("account currency does not match the amount's currency")

	ErrUnknownAccountKind = errors.New("unknown account kind")
)

type CurrencyMismatchError struct {
	AccountID       string
	AccountCurrency string
	AmountCurrency  string
}

func (e *CurrencyMismatchError) Error() string {
	var b strings.Builder
	b.WriteString("account ")
	b.WriteString(e.AccountID)
	b.WriteString(": ")
	b.WriteString(ErrCurrencyMismatch.Error())
	b.WriteString(" (holds ")
	b.WriteString(e.AccountCurrency)
	b.WriteString(", amount in ")
	b.WriteString(e.AmountCurrency)
	b.WriteString(")")
	return b.String()
}

func (e *CurrencyMismatchError) Unwrap() error {
	return ErrCurrencyMismatch
}

type KindError struct {
	Kind         string
	CurrencyCode string
	Err          error
}

func (e *KindError) Error() string {
	var b strings.Builder
	b.WriteString("kind ")
	b.WriteString(e.Kind)
	b.WriteString(" in ")
	b.WriteString(e.CurrencyCode)
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	return b.String()
}

func (e *KindError) Unwrap() error {
	return e.Err
}

type LimitError struct {
	AccountID string
	Amount    int64
	Limit     int64
	Err       error
}

func (e *LimitError) Error() string {
	var b strings.Builder
	b.WriteString("account ")
	b.WriteString(e.AccountID)
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	b.WriteString(" (amount ")
	b.WriteString(strconv.FormatInt(e.Amount, 10))
	b.WriteString(", limit ")
	b.WriteString(strconv.FormatInt(e.Limit, 10))
	b.WriteString(")")
	return b.String()
}

func (e *LimitError) Unwrap() error {
	return e.Err
}

type TierLimitsError struct {
	AccountID string
	Tier      int
}

func (e *TierLimitsError) Error() string {
	var b strings.Builder
	b.WriteString("account ")
	b.WriteString(e.AccountID)
	b.WriteString(": no limits defined for tier ")
	b.WriteString(strconv.Itoa(e.Tier))
	return b.String()
}

func (e *TierLimitsError) Unwrap() error {
	return ErrTierLimitsNotFound
}

type AccountError struct {
	AccountID string
	Err       error
}

func (e *AccountError) Error() string {
	var b strings.Builder
	b.WriteString("account ")
	b.WriteString(e.AccountID)
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	return b.String()
}

func (e *AccountError) Unwrap() error {
	return e.Err
}

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
