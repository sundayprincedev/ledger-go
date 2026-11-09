package accounts

import (
	"fmt"

	"github.com/sundayprincedev/ledger/internal/money"
)

type SavingsAccount struct {
	Account
	Customer
	InterestEarned int64
	InterestRate   float64
}

func (s *SavingsAccount) GenerateSummary() string {
	curr, _ := money.CurrencyExists(s.CurrencyCode)
	balance := money.DenormalizeAmount(s.Balance, curr)
	interest := money.DenormalizeAmount(s.InterestEarned, curr)
	return fmt.Sprintf("Savings Account — %s (ID: %s)\nBalance: %.2f %s | Interest Earned: %.2f %s",
		s.Name,
		s.AccountID,
		balance,
		curr.Name,
		interest,
		curr.Name,
	)
}
