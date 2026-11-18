package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/sundayprincedev/ledger/internal/accounts"
	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

func testStore(t *testing.T) *ledger.AccountStore {
	t.Helper()

	store := ledger.NewAccountStore()
	current := func(id string, tier constants.KYCTier) *accounts.CurrentAccount {
		return &accounts.CurrentAccount{
			Account:  accounts.Account{BaseAccount: accounts.BaseAccount{AccountID: id, CurrencyCode: "NGN"}},
			Customer: accounts.Customer{Tier: tier},
		}
	}

	adds := []error{
		store.AddCurrent(current("CUR-001", constants.TierTwo)),
		store.AddCurrent(current("CUR-003", constants.TierThree)),
		store.AddCurrent(current("CUR-004", constants.TierOne)),
		store.AddCurrent(current("CUR-009", 0)),
		store.AddCurrent(&accounts.CurrentAccount{
			Account:  accounts.Account{BaseAccount: accounts.BaseAccount{AccountID: "CUR-005", CurrencyCode: "USD"}},
			Customer: accounts.Customer{Tier: constants.TierTwo},
		}),
		store.AddHouse(&accounts.House{Account: accounts.Account{BaseAccount: accounts.BaseAccount{AccountID: "HOUSE-NGN", CurrencyCode: "NGN"}}}),
		store.AddSuspense(&accounts.Suspense{Account: accounts.Account{BaseAccount: accounts.BaseAccount{AccountID: "SUSPENSE-NGN", CurrencyCode: "NGN"}}}),
	}
	for _, err := range adds {
		if err != nil {
			t.Fatalf("seeding test store: %v", err)
		}
	}
	return store
}

func TestKYCChecker(t *testing.T) {
	tests := []struct {
		name    string
		intent  func() Intent
		wantErr error
	}{
		{"within limits", testIntent, nil},
		{"amount exactly at the sender limit", func() Intent {
			i := testIntent()
			i.FromAccountID, i.ToAccountID, i.Amount = "CUR-004", "CUR-001", 50000
			return i
		}, nil},
		{"sender over transfer limit", func() Intent {
			i := testIntent()
			i.FromAccountID, i.ToAccountID, i.Amount = "CUR-004", "CUR-001", 60000
			return i
		}, apperrors.ErrTransferLimitExceeded},
		{"receiver over receive limit", func() Intent {
			i := testIntent()
			i.FromAccountID, i.ToAccountID, i.Amount = "CUR-003", "CUR-004", 250000
			return i
		}, apperrors.ErrReceiveLimitExceeded},
		{"unknown sender", func() Intent {
			i := testIntent()
			i.FromAccountID = "NOPE"
			return i
		}, apperrors.ErrAccountNotFound},
		{"unknown receiver", func() Intent {
			i := testIntent()
			i.ToAccountID = "NOPE"
			return i
		}, apperrors.ErrAccountNotFound},
		{"sender without a tier", func() Intent {
			i := testIntent()
			i.FromAccountID = "CUR-009"
			return i
		}, apperrors.ErrTierLimitsNotFound},
		{"house sender is skipped", func() Intent {
			i := testIntent()
			i.Type, i.FromAccountID, i.ToAccountID = OpHouseTransfer, "HOUSE-NGN", "CUR-004"
			return i
		}, nil},
		{"house receiver is skipped", func() Intent {
			i := testIntent()
			i.Type, i.ToAccountID = OpHouseTransfer, "HOUSE-NGN"
			return i
		}, nil},
		{"bulk within limits", func() Intent {
			i := validBulkIntent()
			i.Amount = 60000
			return i
		}, nil},
		{"bulk sender over transfer limit", func() Intent {
			i := validBulkIntent()
			i.FromAccountID, i.Amount = "CUR-004", 60000
			return i
		}, apperrors.ErrTransferLimitExceeded},
	}

	store := testStore(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := kycChecker(store, tc.intent())
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestKYCLimitErrorCarriesAccountAmountAndLimit(t *testing.T) {
	intent := testIntent()
	intent.FromAccountID, intent.ToAccountID, intent.Amount = "CUR-004", "CUR-001", 60000

	err := kycChecker(testStore(t), intent)

	var limitErr *apperrors.LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected a LimitError, got: %v", err)
	}
	if limitErr.AccountID != "CUR-004" || limitErr.Amount != 60000 || limitErr.Limit != 50000 {
		t.Fatalf("unexpected LimitError contents: %+v", limitErr)
	}
}

func TestKYCFailureFailsJobAndReleasesClaim(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t), testWriter(t)))

	over := testIntent()
	over.FromAccountID, over.ToAccountID = "CUR-004", "CUR-001"
	job := NewWriteJob(over, 0)

	jobCh <- job
	close(jobCh)
	wr.Wg.Wait()

	result := <-resultsCh
	if result.Success || !errors.Is(result.Err, apperrors.ErrTransferLimitExceeded) {
		t.Fatalf("expected ErrTransferLimitExceeded, got: %+v", result)
	}

	var pipelineErr *apperrors.JobPipelineError
	if !errors.As(result.Err, &pipelineErr) || pipelineErr.Step != apperrors.PipelineStepKYC {
		t.Fatalf("expected a kyc-step pipeline error, got: %v", result.Err)
	}

	if _, err := registry.LoadJob(job.ID); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("a job rejected after registering must release its claim, got: %v", err)
	}
}
