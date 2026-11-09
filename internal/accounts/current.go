package accounts

import (
	"fmt"

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
	return fmt.Sprintf("Current Account — %s (ID: %s)\nBalance: %.2f %s",
		c.Name,
		c.AccountID,
		balance,
		curr.Name,
	)
}
