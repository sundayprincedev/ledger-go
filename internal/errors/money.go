package apperrors

import (
	"errors"
	"strconv"
	"strings"
)

var (
	ErrAllocateInvalidAmount   = errors.New("allocation amount must be positive")
	ErrAllocateNoRecipients    = errors.New("allocation has no recipients")
	ErrAllocateInvalidWeight   = errors.New("allocation weight must be between zero and the full weight")
	ErrAllocateWeightsDoNotSum = errors.New("allocation weights do not sum to the whole")
	ErrAllocateOverflow        = errors.New("allocation overflows int64")

	ErrSplitInvalidAmount = errors.New("split amount must be positive")
	ErrSplitOverflow      = errors.New("split overflows int64")
	ErrSplitMismatch      = errors.New("split shares do not add up to the amount")
	ErrSplitShareCount    = errors.New("split did not return a house share and a receiver share")
)

type SplitMismatchError struct {
	Amount   int64
	House    int64
	Receiver int64
}

func (e *SplitMismatchError) Error() string {
	var b strings.Builder
	b.WriteString("split of ")
	b.WriteString(strconv.FormatInt(e.Amount, 10))
	b.WriteString(" gave house ")
	b.WriteString(strconv.FormatInt(e.House, 10))
	b.WriteString(" and receiver ")
	b.WriteString(strconv.FormatInt(e.Receiver, 10))
	b.WriteString(": ")
	b.WriteString(ErrSplitMismatch.Error())
	return b.String()
}

func (e *SplitMismatchError) Unwrap() error {
	return ErrSplitMismatch
}

type WeightError struct {
	Index  int
	Weight int64
}

func (e *WeightError) Error() string {
	var b strings.Builder
	b.WriteString("weight ")
	b.WriteString(strconv.FormatInt(e.Weight, 10))
	b.WriteString(" at index ")
	b.WriteString(strconv.Itoa(e.Index))
	b.WriteString(": ")
	b.WriteString(ErrAllocateInvalidWeight.Error())
	return b.String()
}

func (e *WeightError) Unwrap() error {
	return ErrAllocateInvalidWeight
}

type WeightSumError struct {
	Total int64
	Want  int64
}

func (e *WeightSumError) Error() string {
	var b strings.Builder
	b.WriteString(ErrAllocateWeightsDoNotSum.Error())
	b.WriteString(" (total ")
	b.WriteString(strconv.FormatInt(e.Total, 10))
	b.WriteString(", want ")
	b.WriteString(strconv.FormatInt(e.Want, 10))
	b.WriteString(")")
	return b.String()
}

func (e *WeightSumError) Unwrap() error {
	return ErrAllocateWeightsDoNotSum
}
