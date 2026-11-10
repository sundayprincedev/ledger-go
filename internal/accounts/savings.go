package accounts

import (
	"strconv"
	"strings"

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

	var b strings.Builder
	b.WriteString("Savings Account — ")
	b.WriteString(s.Name)
	b.WriteString(" (ID: ")
	b.WriteString(s.AccountID)
	b.WriteString(")\nBalance: ")
	b.WriteString(strconv.FormatFloat(balance, 'f', 2, 64))
	b.WriteString(" ")
	b.WriteString(curr.Name)
	b.WriteString(" | Interest Earned: ")
	b.WriteString(strconv.FormatFloat(interest, 'f', 2, 64))
	b.WriteString(" ")
	b.WriteString(curr.Name)
	return b.String()
}
