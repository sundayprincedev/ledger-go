package jobs

import (
	"errors"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func validBulkIntent() Intent {
	return Intent{
		Type:          OpBulkTransfer,
		FromAccountID: "CUR-001",
		CurrencyCode:  "NGN",
		Amount:        100001,
		Recipients: []Recipient{
			{AccountID: "SAV-001", Weight: 500},
			{AccountID: "CUR-004", Weight: 300},
			{AccountID: "MERCH-001", Weight: 200},
		},
	}
}

func TestIntentValidate(t *testing.T) {
	tests := []struct {
		name      string
		intent    func() Intent
		wantErr   error
		wantStep  apperrors.IntentStep
		wantIndex int
	}{
		{"valid single", testIntent, nil, "", -1},
		{"valid bulk", validBulkIntent, nil, "", -1},
		{"valid house", func() Intent {
			i := testIntent()
			i.Type = OpHouseTransfer
			i.ToAccountID = "HOUSE-NGN"
			return i
		}, nil, "", -1},
		{"unknown type", func() Intent {
			i := testIntent()
			i.Type = "transfr"
			return i
		}, apperrors.ErrUnknownOpType, apperrors.IntentStepType, -1},
		{"empty type", func() Intent {
			i := testIntent()
			i.Type = ""
			return i
		}, apperrors.ErrUnknownOpType, apperrors.IntentStepType, -1},
		{"zero amount", func() Intent {
			i := testIntent()
			i.Amount = 0
			return i
		}, apperrors.ErrIntentInvalidAmount, apperrors.IntentStepAmount, -1},
		{"negative amount", func() Intent {
			i := testIntent()
			i.Amount = -5
			return i
		}, apperrors.ErrIntentInvalidAmount, apperrors.IntentStepAmount, -1},
		{"unknown currency", func() Intent {
			i := testIntent()
			i.CurrencyCode = "XYZ"
			return i
		}, apperrors.ErrUnknownCurrency, apperrors.IntentStepCurrency, -1},
		{"missing from", func() Intent {
			i := testIntent()
			i.FromAccountID = ""
			return i
		}, apperrors.ErrIntentMissingAccount, apperrors.IntentStepAccounts, -1},
		{"missing to", func() Intent {
			i := testIntent()
			i.ToAccountID = ""
			return i
		}, apperrors.ErrIntentMissingAccount, apperrors.IntentStepAccounts, -1},
		{"same account", func() Intent {
			i := testIntent()
			i.ToAccountID = i.FromAccountID
			return i
		}, apperrors.ErrIntentSameAccount, apperrors.IntentStepAccounts, -1},
		{"bulk missing from", func() Intent {
			i := validBulkIntent()
			i.FromAccountID = ""
			return i
		}, apperrors.ErrIntentMissingAccount, apperrors.IntentStepAccounts, -1},
		{"bulk no recipients", func() Intent {
			i := validBulkIntent()
			i.Recipients = nil
			return i
		}, apperrors.ErrIntentNoRecipients, apperrors.IntentStepRecipients, -1},
		{"bulk empty recipient account", func() Intent {
			i := validBulkIntent()
			i.Recipients[1].AccountID = ""
			return i
		}, apperrors.ErrIntentMissingAccount, apperrors.IntentStepRecipients, 1},
		{"bulk recipient is sender", func() Intent {
			i := validBulkIntent()
			i.Recipients[2].AccountID = i.FromAccountID
			return i
		}, apperrors.ErrIntentSameAccount, apperrors.IntentStepRecipients, 2},
		{"bulk zero weight", func() Intent {
			i := validBulkIntent()
			i.Recipients[0].Weight = 0
			return i
		}, apperrors.ErrIntentInvalidWeight, apperrors.IntentStepRecipients, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.intent().Validate()

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected valid intent, got: %v", err)
				}
				return
			}

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got: %v", tc.wantErr, err)
			}

			var validationErr *apperrors.IntentValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected an IntentValidationError, got: %T", err)
			}
			if validationErr.Step != tc.wantStep || validationErr.RecipientIndex != tc.wantIndex {
				t.Fatalf("expected step %q index %d, got step %q index %d", tc.wantStep, tc.wantIndex, validationErr.Step, validationErr.RecipientIndex)
			}
		})
	}
}

func TestGenerateJobID(t *testing.T) {
	base := testIntent()

	if GenerateJobID(base) != GenerateJobID(testIntent()) {
		t.Fatal("same intent must produce the same job id")
	}
	if len(GenerateJobID(base)) != 64 {
		t.Fatalf("expected a 64-char sha256 hex id, got %d chars", len(GenerateJobID(base)))
	}

	changed := testIntent()
	changed.Amount++
	if GenerateJobID(base) == GenerateJobID(changed) {
		t.Fatal("different amounts must produce different job ids")
	}

	noted := testIntent()
	noted.Note = "different note"
	if GenerateJobID(base) != GenerateJobID(noted) {
		t.Fatal("note must not affect the job id")
	}
}
