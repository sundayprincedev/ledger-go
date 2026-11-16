package money

import (
	"errors"
	"math"
	"slices"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func TestAllocateKnownSplits(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		weights []int64
		want    []int64
	}{
		{"exact split", 100000, []int64{500, 300, 200}, []int64{50000, 30000, 20000}},
		{"remainder goes to the largest remainder", 100001, []int64{500, 300, 200}, []int64{50001, 30000, 20000}},
		{"equal remainders favour the lowest index", 1, []int64{200, 200, 200, 200, 200}, []int64{1, 0, 0, 0, 0}},
		{"largest remainder wins over position", 10, []int64{333, 333, 334}, []int64{3, 3, 4}},
		{"two way tie", 1, []int64{500, 500}, []int64{1, 0}},
		{"single recipient takes everything", 777, []int64{1000}, []int64{777}},
		{"zero weight recipient gets nothing", 10, []int64{1000, 0}, []int64{10, 0}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Allocate(tc.amount, tc.weights)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestAllocateAlwaysSumsToAmount(t *testing.T) {
	weightSets := [][]int64{
		{500, 300, 200},
		{333, 333, 334},
		{1, 999},
		{1, 1, 998},
		{200, 200, 200, 200, 200},
		{1000},
		{999, 1},
	}

	for _, weights := range weightSets {
		for amount := int64(1); amount <= 500; amount++ {
			shares, err := Allocate(amount, weights)
			if err != nil {
				t.Fatalf("amount %d weights %v: %v", amount, weights, err)
			}

			var sum int64
			for _, share := range shares {
				if share < 0 {
					t.Fatalf("amount %d weights %v: negative share in %v", amount, weights, shares)
				}
				sum += share
			}
			if sum != amount {
				t.Fatalf("amount %d weights %v: shares %v sum to %d", amount, weights, shares, sum)
			}
		}
	}
}

func TestAllocateRejectsBadInput(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		weights []int64
		wantErr error
	}{
		{"zero amount", 0, []int64{1000}, apperrors.ErrAllocateInvalidAmount},
		{"negative amount", -5, []int64{1000}, apperrors.ErrAllocateInvalidAmount},
		{"no recipients", 100, nil, apperrors.ErrAllocateNoRecipients},
		{"negative weight", 100, []int64{1500, -500}, apperrors.ErrAllocateInvalidWeight},
		{"weight above the whole", 100, []int64{1001}, apperrors.ErrAllocateInvalidWeight},
		{"weights short of the whole", 100, []int64{500, 400}, apperrors.ErrAllocateWeightsDoNotSum},
		{"weights past the whole", 100, []int64{600, 600}, apperrors.ErrAllocateWeightsDoNotSum},
		{"weights that would wrap an int64 sum", 100, []int64{math.MaxInt64, math.MaxInt64, 1002}, apperrors.ErrAllocateInvalidWeight},
		{"amount large enough to overflow", math.MaxInt64, []int64{1000}, apperrors.ErrAllocateOverflow},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			shares, err := Allocate(tc.amount, tc.weights)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got: %v", tc.wantErr, err)
			}
			if shares != nil {
				t.Fatalf("expected no shares on error, got %v", shares)
			}
		})
	}
}

func TestAllocateErrorsCarryDetail(t *testing.T) {
	_, err := Allocate(100, []int64{500, 400})
	var sumErr *apperrors.WeightSumError
	if !errors.As(err, &sumErr) || sumErr.Total != 900 || sumErr.Want != FullWeightUnits {
		t.Fatalf("expected WeightSumError{900, %d}, got: %v", FullWeightUnits, err)
	}

	_, err = Allocate(100, []int64{500, -1, 501})
	var weightErr *apperrors.WeightError
	if !errors.As(err, &weightErr) || weightErr.Index != 1 || weightErr.Weight != -1 {
		t.Fatalf("expected WeightError{1, -1}, got: %v", err)
	}
}

func TestAllocateLargestAmountThatFits(t *testing.T) {
	amount := math.MaxInt64 / FullWeightUnits

	shares, err := Allocate(amount, []int64{1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shares) != 1 || shares[0] != amount {
		t.Fatalf("expected [%d], got %v", amount, shares)
	}
}
