package ledger

import (
	"encoding/json"
	"time"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
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
		return Entry{}, &apperrors.EntryDecodeError{Err: err}
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
		return &apperrors.EntryValidationError{Step: apperrors.StepEntryID, LineIndex: -1, Err: apperrors.ErrEmptyEntryID}
	}
	if len(e.Lines) < 2 {
		return &apperrors.EntryValidationError{Step: apperrors.StepLineCount, LineIndex: -1, Err: apperrors.ErrTooFewLines}
	}

	totals := make(map[string]currencyTotals)

	for i, line := range e.Lines {
		if line.AccountID == "" {
			return &apperrors.EntryValidationError{Step: apperrors.StepAccountID, LineIndex: i, Err: apperrors.ErrEmptyAccountID}
		}
		if line.Amount <= 0 {
			return &apperrors.EntryValidationError{Step: apperrors.StepAmount, LineIndex: i, Err: apperrors.ErrLineInvalidAmount}
		}
		if _, ok := money.CurrencyExists(line.CurrencyCode); !ok {
			return &apperrors.EntryValidationError{
				Step:      apperrors.StepCurrency,
				LineIndex: i,
				Err:       &apperrors.CurrencyError{Code: line.CurrencyCode},
			}
		}

		t := totals[line.CurrencyCode]
		switch line.Direction {
		case Debit:
			sum, err := addChecked(t.Debits, line.Amount)
			if err != nil {
				return &apperrors.EntryValidationError{Step: apperrors.StepOverflow, LineIndex: i, Err: err}
			}
			t.Debits = sum
		case Credit:
			sum, err := addChecked(t.Credits, line.Amount)
			if err != nil {
				return &apperrors.EntryValidationError{Step: apperrors.StepOverflow, LineIndex: i, Err: err}
			}
			t.Credits = sum
		default:
			return &apperrors.EntryValidationError{
				Step:      apperrors.StepDirection,
				LineIndex: i,
				Err:       &apperrors.DirectionError{Direction: string(line.Direction)},
			}
		}
		totals[line.CurrencyCode] = t
	}

	for currencyCode, t := range totals {
		if t.Debits != t.Credits {
			return &apperrors.EntryValidationError{
				Step:      apperrors.StepBalance,
				LineIndex: -1,
				Err:       &apperrors.BalanceError{CurrencyCode: currencyCode, Debits: t.Debits, Credits: t.Credits},
			}
		}
	}

	return nil
}

func addChecked(a, b int64) (int64, error) {
	sum := a + b
	if sum < a {
		return 0, apperrors.ErrAmountOverflow
	}
	return sum, nil
}
