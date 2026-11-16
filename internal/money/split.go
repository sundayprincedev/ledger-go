package money

import (
	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func SplitHouseCut(amount int64) ([]int64, error) {
	if amount <= 0 {
		return nil, apperrors.ErrSplitInvalidAmount
	}

	cutShares := make([]int64, 2)

	if amount <= constants.HouseFeeThreshold {
		cutShares[0] = 0
		cutShares[1] = amount

		return verifySplit(amount, cutShares[0], cutShares[1])
	}

	weighted, ok := multiplyChecked(amount, constants.HouseFeeWeight)
	if !ok {
		return nil, apperrors.ErrSplitOverflow
	}
	pct := weighted / 1000 //Due to weighted 20 figure.

	cutShares[0] = pct
	cutShares[1] = amount - pct

	return verifySplit(amount, cutShares[0], cutShares[1])
}

func verifySplit(amount, house, receiver int64) ([]int64, error) {
	if house < 0 || receiver < 0 || house+receiver != amount {
		return nil, &apperrors.SplitMismatchError{Amount: amount, House: house, Receiver: receiver}
	}
	return []int64{house, receiver}, nil
}
