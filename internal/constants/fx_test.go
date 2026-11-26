package constants_test

import (
	"testing"

	"github.com/sundayprincedev/ledger/internal/constants"
	"github.com/sundayprincedev/ledger/internal/money"
)

func TestEveryKnownCurrencyHasARate(t *testing.T) {
	for code := range money.Currencies {
		if rate, ok := constants.FXPerBase[code]; !ok || rate <= 0 {
			t.Fatalf("currency %s needs a positive rate, got %d (present: %v)", code, rate, ok)
		}
	}
}

func TestEveryRateBelongsToAKnownCurrency(t *testing.T) {
	for code := range constants.FXPerBase {
		if _, ok := money.CurrencyExists(code); !ok {
			t.Fatalf("rate for %s has no matching currency", code)
		}
	}
}

func TestTheBaseCurrencyIsExactlyOneUnit(t *testing.T) {
	if got := constants.FXPerBase[constants.FXBaseCurrency]; got != constants.FXRateScale {
		t.Fatalf("the base currency must be 1.0, got %d of scale %d", got, constants.FXRateScale)
	}
}
