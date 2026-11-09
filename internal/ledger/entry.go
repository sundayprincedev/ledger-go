package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sundayprincedev/ledger/internal/money"
)

type Direction string

const (
	Debit  Direction = "DEBIT"
	Credit Direction = "CREDIT"
)

type Reference struct {
	JobID                 string `json:"job_id"`
	CounterpartyAccountID string `json:"counterparty_account_id"`
}

type Line struct {
	AccountID    string    `json:"account_id"`
	Direction    Direction `json:"direction"`
	Amount       int64     `json:"amount"`
	CurrencyCode string    `json:"currency"`
	Reference    Reference `json:"reference"`
}

type Entry struct {
	ID        string
	Timestamp time.Time
	Lines     []Line
}

type entryPayload struct {
	ID    string `json:"id"`
	Lines []Line `json:"lines"`
}

func (e *Entry) Marshal() ([]byte, error) {
	return json.Marshal(entryPayload{ID: e.ID, Lines: e.Lines})
}

func EntryFromPayload(payload []byte, timestamp time.Time) (Entry, error) {
	var p entryPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return Entry{}, fmt.Errorf("decode entry: %w", err)
	}
	e := Entry{ID: p.ID, Lines: p.Lines, Timestamp: timestamp}
	if err := e.Validate(); err != nil {
		return Entry{}, err
	}
	return e, nil
}

type currencyTotals struct {
	Debits  int64
	Credits int64
}

func (e *Entry) Validate() error {
	if e.ID == "" {
		return ErrEmptyEntryID
	}
	if len(e.Lines) < 2 {
		return ErrTooFewLines
	}

	totals := make(map[string]currencyTotals)

	for _, line := range e.Lines {
		if line.AccountID == "" {
			return ErrEmptyAccountID
		}
		if line.Amount <= 0 {
			return ErrInvalidAmount
		}
		if _, ok := money.CurrencyExists(line.CurrencyCode); !ok {
			return fmt.Errorf("%w: %s", ErrUnknownCurrency, line.CurrencyCode)
		}

		t := totals[line.CurrencyCode]
		switch line.Direction {
		case Debit:
			sum, err := addChecked(t.Debits, line.Amount)
			if err != nil {
				return err
			}
			t.Debits = sum
		case Credit:
			sum, err := addChecked(t.Credits, line.Amount)
			if err != nil {
				return err
			}
			t.Credits = sum
		default:
			return fmt.Errorf("%w: %s", ErrUnknownDirection, line.Direction)
		}
		totals[line.CurrencyCode] = t
	}

	for currencyCode, t := range totals {
		if t.Debits != t.Credits {
			return fmt.Errorf("%w: %s debits=%d credits=%d", ErrUnbalancedEntry, currencyCode, t.Debits, t.Credits)
		}
	}

	return nil
}

func addChecked(a, b int64) (int64, error) {
	sum := a + b
	if sum < a {
		return 0, ErrAmountOverflow
	}
	return sum, nil
}
