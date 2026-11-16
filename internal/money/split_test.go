package money

import (
	"errors"
	"math"
	"testing"

	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func TestSplitHouseCutRejectsNonPositiveAmount(t *testing.T) {
	for _, amount := range []int64{0, -1, math.MinInt64} {
		shares, err := SplitHouseCut(amount)
		if !errors.Is(err, apperrors.ErrSplitInvalidAmount) || shares != nil {
			t.Fatalf("amount %d: expected ErrSplitInvalidAmount and no shares, got %v, %v", amount, shares, err)
		}
	}
}

func TestSplitHouseCutOverflow(t *testing.T) {
	shares, err := SplitHouseCut(math.MaxInt64)
	if !errors.Is(err, apperrors.ErrSplitOverflow) || shares != nil {
		t.Fatalf("expected ErrSplitOverflow and no shares, got %v, %v", shares, err)
	}
}

func TestSplitHouseCutLargestAmountThatFits(t *testing.T) {
	amount := math.MaxInt64 / constants.HouseFeeWeight

	shares, err := SplitHouseCut(amount)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shares[0]+shares[1] != amount {
		t.Fatalf("shares %v do not add up to %d", shares, amount)
	}
}

func TestSplitHouseCutBelowThresholdGivesTheHouseNothing(t *testing.T) {
	for _, amount := range []int64{1, 5000, constants.HouseFeeThreshold} {
		shares, err := SplitHouseCut(amount)
		if err != nil {
			t.Fatalf("amount %d: unexpected error: %v", amount, err)
		}
		if len(shares) != 2 || shares[0] != 0 || shares[1] != amount {
			t.Fatalf("amount %d: expected [0 %d], got %v", amount, amount, shares)
		}
	}
}

func TestSplitHouseCutAboveThresholdAddsUp(t *testing.T) {
	for _, amount := range []int64{constants.HouseFeeThreshold + 1, 123457, 5000000} {
		shares, err := SplitHouseCut(amount)
		if err != nil {
			t.Fatalf("amount %d: unexpected error: %v", amount, err)
		}
		if len(shares) != 2 {
			t.Fatalf("amount %d: expected two shares, got %v", amount, shares)
		}
		if shares[0] <= 0 || shares[1] <= 0 || shares[0] >= amount {
			t.Fatalf("amount %d: expected both shares positive, got %v", amount, shares)
		}
		if shares[0]+shares[1] != amount {
			t.Fatalf("amount %d: shares %v do not add up", amount, shares)
		}
	}
}

func TestVerifySplitCatchesBadShares(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		house    int64
		receiver int64
	}{
		{"shares short of the amount", 100, 0, 1},
		{"shares past the amount", 100, 60, 60},
		{"negative house share", 100, -10, 110},
		{"negative receiver share", 100, 110, -10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			shares, err := verifySplit(tc.amount, tc.house, tc.receiver)
			if !errors.Is(err, apperrors.ErrSplitMismatch) || shares != nil {
				t.Fatalf("expected ErrSplitMismatch and no shares, got %v, %v", shares, err)
			}

			var mismatch *apperrors.SplitMismatchError
			if !errors.As(err, &mismatch) || mismatch.Amount != tc.amount || mismatch.House != tc.house || mismatch.Receiver != tc.receiver {
				t.Fatalf("expected the mismatch to carry its inputs, got: %v", err)
			}
		})
	}
}
