package accounts

import (
	"strconv"
	"strings"

	"github.com/sundayprincedev/ledger/internal/money"
)

type CurrentAccount struct {
	Account
	Customer
	OverdraftLimit int64
}

func (c *CurrentAccount) GenerateSummary() string {
	curr, _ := money.CurrencyExists(c.CurrencyCode)
	balance := money.DenormalizeAmount(c.Balance, curr)

	var b strings.Builder
	b.WriteString("Current Account — ")
	b.WriteString(c.Name)
	b.WriteString(" (ID: ")
	b.WriteString(c.AccountID)
	b.WriteString(")\nBalance: ")
	b.WriteString(strconv.FormatFloat(balance, 'f', 2, 64))
	b.WriteString(" ")
	b.WriteString(curr.Name)
	return b.String()
}
