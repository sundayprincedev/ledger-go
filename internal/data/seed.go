package data

import (
	"encoding/json"
	"fmt"
	"os"
)

const LedgerLogFile = "ledger.log"

type SeedAccount struct {
	AccountID      string  `json:"account_id"`
	Kind           string  `json:"kind"`
	CurrencyCode   string  `json:"currency"`
	Balance        int64   `json:"balance"`
	CustomerID     string  `json:"customer_id,omitempty"`
	CustomerName   string  `json:"customer_name,omitempty"`
	MerchantID     string  `json:"merchant_id,omitempty"`
	MerchantName   string  `json:"merchant_name,omitempty"`
	OverdraftLimit int64   `json:"overdraft_limit,omitempty"`
	InterestRate   float64 `json:"interest_rate,omitempty"`
	Tier           int     `json:"tier,omitempty"`
}

type SeedRecipient struct {
	AccountID string `json:"account_id"`
	Weight    int64  `json:"weight"`
}

type SeedOperation struct {
	Type          string          `json:"type"`
	AccountID     string          `json:"account_id,omitempty"`
	FromAccountID string          `json:"from_account_id,omitempty"`
	ToAccountID   string          `json:"to_account_id,omitempty"`
	CurrencyCode  string          `json:"currency"`
	Amount        int64           `json:"amount"`
	Recipients    []SeedRecipient `json:"recipients,omitempty"`
	Note          string          `json:"note,omitempty"`
}

type SeedData struct {
	Accounts   []SeedAccount   `json:"accounts"`
	Operations []SeedOperation `json:"operations"`
}

func Load(path string) (SeedData, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return SeedData{}, fmt.Errorf("read seed file: %w", err)
	}

	var data SeedData
	if err := json.Unmarshal(file, &data); err != nil {
		return SeedData{}, fmt.Errorf("parse seed data: %w", err)
	}

	return data, nil
}
