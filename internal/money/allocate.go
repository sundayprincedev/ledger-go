package money

import (
	"sort"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

const FullWeightUnits int64 = 1000

func Allocate(amount int64, weights []int64) ([]int64, error) {
	if amount <= 0 {
		return nil, apperrors.ErrAllocateInvalidAmount
	}
	if len(weights) == 0 {
		return nil, apperrors.ErrAllocateNoRecipients
	}

	var totalWeight int64
	for index, weight := range weights {
		if weight < 0 || weight > FullWeightUnits {
			return nil, &apperrors.WeightError{Index: index, Weight: weight}
		}
		totalWeight += weight
	}
	if totalWeight != FullWeightUnits {
		return nil, &apperrors.WeightSumError{Total: totalWeight, Want: FullWeightUnits}
	}

	shares := make([]int64, len(weights))
	remainders := make([]int64, len(weights))
	var allocated int64

	for index, weight := range weights {
		weighted, ok := multiplyChecked(weight, amount)
		if !ok {
			return nil, apperrors.ErrAllocateOverflow
		}
		shares[index] = weighted / FullWeightUnits
		remainders[index] = weighted % FullWeightUnits
		allocated += shares[index]
	}

	order := make([]int, len(weights))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		if remainders[order[a]] == remainders[order[b]] {
			return order[a] < order[b]
		}
		return remainders[order[a]] > remainders[order[b]]
	})

	leftover := int(amount - allocated)
	for _, index := range order[:leftover] {
		shares[index]++
	}

	return shares, nil
}

func multiplyChecked(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	result := a * b
	if result/a != b {
		return 0, false
	}
	return result, true
}
