package ledger

import (
	"math"
	"sync"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

type balanceKey struct {
	accountID    string
	currencyCode string
}

type Balances struct {
	mu sync.RWMutex
	m  map[balanceKey]int64
}

func NewBalances() *Balances {
	return &Balances{m: make(map[balanceKey]int64)}
}

func (b *Balances) Set(accountID, currencyCode string, amount int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.m[balanceKey{accountID, currencyCode}] = amount
}

func (b *Balances) Get(accountID, currencyCode string) int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.m[balanceKey{accountID, currencyCode}]
}

func (b *Balances) Apply(e Entry) error {
	deltas := make(map[balanceKey]int64, len(e.Lines))
	for _, line := range e.Lines {
		key := balanceKey{line.AccountID, line.CurrencyCode}
		amount := line.Amount
		if line.Direction == Debit {
			amount = -amount
		}
		next, ok := addSigned(deltas[key], amount)
		if !ok {
			return &apperrors.BalanceOverflowError{AccountID: key.accountID, CurrencyCode: key.currencyCode}
		}
		deltas[key] = next
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	updated := make(map[balanceKey]int64, len(deltas))
	for key, delta := range deltas {
		next, ok := addSigned(b.m[key], delta)
		if !ok {
			return &apperrors.BalanceOverflowError{AccountID: key.accountID, CurrencyCode: key.currencyCode}
		}
		updated[key] = next
	}
	for key, next := range updated {
		b.m[key] = next
	}
	return nil
}

func addSigned(a, b int64) (int64, bool) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, false
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, false
	}
	return a + b, true
}
