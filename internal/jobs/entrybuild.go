package jobs

import (
	"time"

	"github.com/sundayprincedev/ledger/internal/accounts"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

func buildDirectEntry(store *ledger.AccountStore, jobID string, intent Intent, splitCut []int64) (ledger.Entry, error) {
	totalDebitAmount := splitCut[0] + splitCut[1]

	entryLines := make([]ledger.Line, 0, 6)

	suspenseProfile, suspenseErr := store.FindByKind(accounts.KindSuspense, intent.CurrencyCode)
	if suspenseErr != nil {
		return ledger.Entry{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepBuildEntry, Err: suspenseErr}
	}

	fromSuspenseAccount := &accounts.Suspense{
		Account: accounts.Account{
			BaseAccount: accounts.BaseAccount{
				AccountID:    suspenseProfile.ID,
				Kind:         suspenseProfile.Kind,
				CurrencyCode: suspenseProfile.CurrencyCode,
			},
		},
	}

	fromUserDebitedLine := ledger.Line{
		AccountID:    intent.FromAccountID,
		Direction:    ledger.Debit,
		Amount:       totalDebitAmount,
		CurrencyCode: intent.CurrencyCode,
		Reference:    ledger.Reference{JobID: jobID, CounterpartyAccountID: fromSuspenseAccount.AccountID},
	}
	entryLines = append(entryLines, fromUserDebitedLine)

	suspenseCreditedFromUserLine := ledger.Line{
		AccountID:    fromSuspenseAccount.AccountID,
		Direction:    ledger.Credit,
		Amount:       totalDebitAmount,
		CurrencyCode: intent.CurrencyCode,
		Reference:    ledger.Reference{JobID: jobID, CounterpartyAccountID: intent.FromAccountID},
	}
	entryLines = append(entryLines, suspenseCreditedFromUserLine)

	if splitCut[0] > 0 {
		houseProfile, houseErr := store.FindByKind(accounts.KindHouse, intent.CurrencyCode)
		if houseErr != nil {
			return ledger.Entry{}, &apperrors.JobPipelineError{JobID: jobID, Step: apperrors.PipelineStepBuildEntry, Err: houseErr}
		}

		toHouseAccount := &accounts.House{
			Account: accounts.Account{
				BaseAccount: accounts.BaseAccount{
					AccountID:    houseProfile.ID,
					Kind:         houseProfile.Kind,
					CurrencyCode: houseProfile.CurrencyCode,
				},
			},
		}

		houseCreditedFromSuspenseLine := ledger.Line{
			AccountID:    toHouseAccount.AccountID,
			Direction:    ledger.Credit,
			Amount:       splitCut[0],
			CurrencyCode: intent.CurrencyCode,
			Reference:    ledger.Reference{JobID: jobID, CounterpartyAccountID: fromSuspenseAccount.AccountID},
		}
		entryLines = append(entryLines, houseCreditedFromSuspenseLine)

		suspenseDebitedToHouseLine := ledger.Line{
			AccountID:    fromSuspenseAccount.AccountID,
			Direction:    ledger.Debit,
			Amount:       splitCut[0],
			CurrencyCode: intent.CurrencyCode,
			Reference:    ledger.Reference{JobID: jobID, CounterpartyAccountID: toHouseAccount.AccountID},
		}
		entryLines = append(entryLines, suspenseDebitedToHouseLine)
	}

	toUserCreditedFromSuspenseLine := ledger.Line{
		AccountID:    intent.ToAccountID,
		Direction:    ledger.Credit,
		Amount:       splitCut[1],
		CurrencyCode: intent.CurrencyCode,
		Reference:    ledger.Reference{JobID: jobID, CounterpartyAccountID: fromSuspenseAccount.AccountID},
	}
	entryLines = append(entryLines, toUserCreditedFromSuspenseLine)

	suspenseDebitedToUserLine := ledger.Line{
		AccountID:    fromSuspenseAccount.AccountID,
		Direction:    ledger.Debit,
		Amount:       splitCut[1],
		CurrencyCode: intent.CurrencyCode,
		Reference:    ledger.Reference{JobID: jobID, CounterpartyAccountID: intent.ToAccountID},
	}
	entryLines = append(entryLines, suspenseDebitedToUserLine)

	return ledger.Entry{
		ID:        jobID,
		Timestamp: time.Now(),
		Lines:     entryLines,
	}, nil
}
