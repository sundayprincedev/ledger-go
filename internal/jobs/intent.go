package jobs

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/money"
)

type OpType string

const (
	OpSingleTransfer OpType = "single_transfer"
	OpBulkTransfer   OpType = "bulk_transfer"
	OpHouseTransfer  OpType = "house_transfer"
)

type Recipient struct {
	AccountID string
	Weight    int64
}

type Intent struct {
	Type          OpType
	AccountID     string
	FromAccountID string
	ToAccountID   string
	CurrencyCode  string
	Amount        int64
	Recipients    []Recipient
	Note          string
}

func GenerateJobID(intent Intent) string {
	var b strings.Builder
	b.WriteString(string(intent.Type))
	b.WriteString("|")
	b.WriteString(intent.AccountID)
	b.WriteString("|")
	b.WriteString(intent.FromAccountID)
	b.WriteString("|")
	b.WriteString(intent.ToAccountID)
	b.WriteString("|")
	b.WriteString(intent.CurrencyCode)
	b.WriteString("|")
	b.WriteString(strconv.FormatInt(intent.Amount, 10))
	b.WriteString("|")
	for _, r := range intent.Recipients {
		b.WriteString(r.AccountID)
		b.WriteString(":")
		b.WriteString(strconv.FormatInt(r.Weight, 10))
		b.WriteString(",")
	}

	data := []byte(b.String())
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (t OpType) valid() bool {
	switch t {
	case OpSingleTransfer, OpBulkTransfer, OpHouseTransfer:
		return true
	}
	return false
}

func (i Intent) Validate() error {
	if !i.Type.valid() {
		return &apperrors.IntentValidationError{
			Step:           apperrors.IntentStepType,
			RecipientIndex: -1,
			Err:            &apperrors.OpTypeError{Type: string(i.Type)},
		}
	}
	if i.Amount <= 0 {
		return &apperrors.IntentValidationError{
			Step:           apperrors.IntentStepAmount,
			RecipientIndex: -1,
			Err:            apperrors.ErrIntentInvalidAmount,
		}
	}
	if _, ok := money.CurrencyExists(i.CurrencyCode); !ok {
		return &apperrors.IntentValidationError{
			Step:           apperrors.IntentStepCurrency,
			RecipientIndex: -1,
			Err:            &apperrors.CurrencyError{Code: i.CurrencyCode},
		}
	}

	if i.Type == OpBulkTransfer {
		return i.validateRecipients()
	}
	return i.validateAccountPair()
}

func (i Intent) validateAccountPair() error {
	if i.FromAccountID == "" || i.ToAccountID == "" {
		return &apperrors.IntentValidationError{
			Step:           apperrors.IntentStepAccounts,
			RecipientIndex: -1,
			Err:            apperrors.ErrIntentMissingAccount,
		}
	}
	if i.FromAccountID == i.ToAccountID {
		return &apperrors.IntentValidationError{
			Step:           apperrors.IntentStepAccounts,
			RecipientIndex: -1,
			Err:            apperrors.ErrIntentSameAccount,
		}
	}
	return nil
}

func (i Intent) validateRecipients() error {
	if i.FromAccountID == "" {
		return &apperrors.IntentValidationError{
			Step:           apperrors.IntentStepAccounts,
			RecipientIndex: -1,
			Err:            apperrors.ErrIntentMissingAccount,
		}
	}
	if len(i.Recipients) == 0 {
		return &apperrors.IntentValidationError{
			Step:           apperrors.IntentStepRecipients,
			RecipientIndex: -1,
			Err:            apperrors.ErrIntentNoRecipients,
		}
	}
	for idx, r := range i.Recipients {
		if r.AccountID == "" {
			return &apperrors.IntentValidationError{
				Step:           apperrors.IntentStepRecipients,
				RecipientIndex: idx,
				Err:            apperrors.ErrIntentMissingAccount,
			}
		}
		if r.AccountID == i.FromAccountID {
			return &apperrors.IntentValidationError{
				Step:           apperrors.IntentStepRecipients,
				RecipientIndex: idx,
				Err:            apperrors.ErrIntentSameAccount,
			}
		}
		if r.Weight <= 0 {
			return &apperrors.IntentValidationError{
				Step:           apperrors.IntentStepRecipients,
				RecipientIndex: idx,
				Err:            apperrors.ErrIntentInvalidWeight,
			}
		}
	}
	return nil
}

func NewWriteJob(intent Intent, timeout time.Duration) *Job {
	return &Job{
		ID:        GenerateJobID(intent),
		Payload:   intent,
		Timeout:   timeout,
		CreatedAt: time.Now().UTC(),
	}
}
