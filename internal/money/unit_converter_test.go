package money

import (
	"errors"
	"math"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func TestUnitConverterKnownAmounts(t *testing.T) {
	tests := []struct {
		name     string
		from, to string
		amount   int64
		want     int64
	}{
		{"NGN to USD", "NGN", "USD", 150000, 100},
		{"USD to NGN", "USD", "NGN", 100, 150000},
		{"USD to JPY drops the decimals", "USD", "JPY", 100, 150},
		{"JPY to USD adds the decimals", "JPY", "USD", 150, 100},
		{"cross rate through the base", "NGN", "GBP", 150000, 79},
		{"half up on an exact tie", "NGN", "USD", 2250, 2},
		{"below the tie rounds down", "NGN", "USD", 2249, 1},
		{"zero stays zero", "NGN", "USD", 0, 0},
		{"same currency is unchanged", "USD", "USD", 12345, 12345},
		{"a large amount keeps its precision", "NGN", "USD", 10_000_000_000, 6_666_667},
		{"the largest amount does not overflow midway", "NGN", "USD", math.MaxInt64, 6_148_914_691_236_517},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UnitConverter(tc.from, tc.to, tc.amount)
			if err != nil {
				t.Fatalf("UnitConverter: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %d, got %d", tc.want, got)
			}
		})
	}
}

func TestUnitConverterRejectsAResultThatDoesNotFitInt64(t *testing.T) {
	_, err := UnitConverter("USD", "NGN", math.MaxInt64)

	if !errors.Is(err, apperrors.ErrConvertOverflow) {
		t.Fatalf("expected ErrConvertOverflow, got %v", err)
	}
}

func TestUnitConverterRejectsUnknownCurrenciesAndNegativeAmounts(t *testing.T) {
	if _, err := UnitConverter("XXX", "USD", 100); !errors.Is(err, apperrors.ErrUnknownCurrency) {
		t.Fatalf("an unknown source must be rejected, got %v", err)
	}
	if _, err := UnitConverter("USD", "XXX", 100); !errors.Is(err, apperrors.ErrUnknownCurrency) {
		t.Fatalf("an unknown target must be rejected, got %v", err)
	}
	if _, err := UnitConverter("USD", "NGN", -1); !errors.Is(err, apperrors.ErrConvertInvalidAmount) {
		t.Fatalf("a negative amount must be rejected, got %v", err)
	}
}
