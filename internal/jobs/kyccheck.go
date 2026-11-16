package jobs

import (
	"github.com/sundayprincedev/ledger/internal/accounts"
	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

type kycRole int

const (
	kycSender kycRole = iota
	kycReceiver
)

func kycChecker(store *ledger.AccountStore, intent Intent) error {
	if err := checkAccountLimit(store, intent.FromAccountID, intent.Amount, kycSender); err != nil {
		return err
	}

	if intent.Type == OpBulkTransfer {
		// TODO: once Allocate produces each recipient's share, call checkAccountLimit with kycReceiver for every recipient
		return nil
	}

	return checkAccountLimit(store, intent.ToAccountID, intent.Amount, kycReceiver)
}

func checkAccountLimit(store *ledger.AccountStore, accountID string, amount int64, role kycRole) error {
	profile, err := store.Lookup(accountID)
	if err != nil {
		return err
	}

	if profile.Kind == accounts.KindHouse || profile.Kind == accounts.KindSuspense {
		return nil
	}

	limits, ok := constants.LimitsFor(profile.Tier)
	if !ok {
		return &apperrors.TierLimitsError{AccountID: profile.ID, Tier: int(profile.Tier)}
	}

	if role == kycSender && amount > limits.SingleTransferLimit {
		return &apperrors.LimitError{
			AccountID: profile.ID,
			Amount:    amount,
			Limit:     limits.SingleTransferLimit,
			Err:       apperrors.ErrTransferLimitExceeded,
		}
	}
	if role == kycReceiver && amount > limits.SingleReceiveLimit {
		return &apperrors.LimitError{
			AccountID: profile.ID,
			Amount:    amount,
			Limit:     limits.SingleReceiveLimit,
			Err:       apperrors.ErrReceiveLimitExceeded,
		}
	}

	return nil
}
