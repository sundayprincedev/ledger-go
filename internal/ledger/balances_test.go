package ledger

import (
	"errors"
	"math"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func transferEntry(amount int64) Entry {
	return Entry{ID: "job", Lines: []Line{
		{AccountID: "A", Direction: Debit, Amount: amount, CurrencyCode: "NGN"},
		{AccountID: "B", Direction: Credit, Amount: amount, CurrencyCode: "NGN"},
	}}
}

func TestApplyMovesBalancesCreditNormal(t *testing.T) {
	b := NewBalances()
	b.Set("A", "NGN", 1000)
	b.Set("B", "NGN", 50)

	if err := b.Apply(transferEntry(300)); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if got := b.Get("A", "NGN"); got != 700 {
		t.Fatalf("sender: want 700, got %d", got)
	}
	if got := b.Get("B", "NGN"); got != 350 {
		t.Fatalf("receiver: want 350, got %d", got)
	}
}

func TestApplyNetsSeveralLinesOnOneAccount(t *testing.T) {
	b := NewBalances()
	e := Entry{ID: "job", Lines: []Line{
		{AccountID: "S", Direction: Credit, Amount: 100, CurrencyCode: "NGN"},
		{AccountID: "S", Direction: Debit, Amount: 100, CurrencyCode: "NGN"},
		{AccountID: "A", Direction: Debit, Amount: 100, CurrencyCode: "NGN"},
		{AccountID: "B", Direction: Credit, Amount: 100, CurrencyCode: "NGN"},
	}}

	if err := b.Apply(e); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := b.Get("S", "NGN"); got != 0 {
		t.Fatalf("pass-through account must end at zero, got %d", got)
	}
}

func TestApplyChangesNothingWhenAnyBalanceOverflows(t *testing.T) {
	b := NewBalances()
	b.Set("A", "NGN", 1000)
	b.Set("B", "NGN", math.MaxInt64-10)

	err := b.Apply(transferEntry(300))

	if !errors.Is(err, apperrors.ErrBalanceOverflow) {
		t.Fatalf("expected ErrBalanceOverflow, got %v", err)
	}
	if b.Get("A", "NGN") != 1000 || b.Get("B", "NGN") != math.MaxInt64-10 {
		t.Fatal("a failed apply must leave every balance untouched")
	}
}
