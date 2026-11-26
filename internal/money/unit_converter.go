package money

import (
	"math/big"

	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func UnitConverter(fromCode, toCode string, amount int64) (int64, error) {
	if currCheckErr := CurrExistCheck(fromCode, toCode); currCheckErr != nil {
		return 0, currCheckErr
	}

	if amount < 0 {
		return 0, apperrors.ErrConvertInvalidAmount
	}

	if fromCode == toCode {
		return amount, nil
	}

	fromCurr, toCurr := Currencies[fromCode], Currencies[toCode]
	fxBaseFrom, fxBaseTo := constants.FXPerBase[fromCode], constants.FXPerBase[toCode]

	numerator := big.NewInt(amount)
	numerator.Mul(numerator, big.NewInt(fxBaseTo))
	numerator.Mul(numerator, pow10(toCurr.DecimalPlaces))

	denominator := big.NewInt(fxBaseFrom)
	denominator.Mul(denominator, pow10(fromCurr.DecimalPlaces))

	numerator.Lsh(numerator, 1)
	numerator.Add(numerator, denominator)
	denominator.Lsh(denominator, 1)
	amountInTarget := numerator.Quo(numerator, denominator)

	if !amountInTarget.IsInt64() {
		return 0, apperrors.ErrConvertOverflow
	}
	return amountInTarget.Int64(), nil
}

func pow10(exponent int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
}

func CurrExistCheck(codes ...string) error {
	for _, code := range codes {
		if _, ok := CurrencyExists(code); !ok {
			return &apperrors.CurrencyError{Code: code}
		}
	}
	return nil
}
