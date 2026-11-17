package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sundayprincedev/ledger/internal/accounts"
	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

func testIntent() Intent {
	return Intent{
		Type:          OpSingleTransfer,
		FromAccountID: "CUR-001",
		ToAccountID:   "CUR-004",
		CurrencyCode:  "NGN",
		Amount:        60000,
	}
}

func TestWriteWorkRunsThroughPool(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 2)
	errCh := make(chan error, 4)
	resultsCh := make(chan JobResult, 2)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t)))

	first := NewWriteJob(testIntent(), 0)
	second := NewWriteJob(testIntent(), 0)
	if first.ID != second.ID {
		t.Fatalf("same intent produced different job ids: %s vs %s", first.ID, second.ID)
	}

	jobCh <- first
	jobCh <- second
	close(jobCh)
	wr.Wg.Wait()

	firstResult := <-resultsCh
	if !firstResult.Success {
		t.Fatalf("fresh job with zero timeout should succeed, got: %v", firstResult.Err)
	}

	lookup, err := registry.LoadJob(first.ID)
	if err != nil || lookup.State != ledger.StateCompleted {
		t.Fatalf("registry entry should be completed after a successful run, got %v (err: %v)", lookup.State, err)
	}

	secondResult := <-resultsCh
	if secondResult.Success {
		t.Fatal("duplicate of a completed job should not succeed")
	}
	if !errors.Is(secondResult.Err, apperrors.ErrJobAlreadyCompleted) {
		t.Fatalf("expected ErrJobAlreadyCompleted, got: %v", secondResult.Err)
	}

	var pipelineErr *apperrors.JobPipelineError
	if !errors.As(secondResult.Err, &pipelineErr) || pipelineErr.Step != apperrors.PipelineStepRegister {
		t.Fatalf("expected a register-step pipeline error, got: %v", secondResult.Err)
	}
}

func TestInvalidIntentIsRejectedBeforeRegister(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t)))

	bad := testIntent()
	bad.Type = "transfr"
	job := NewWriteJob(bad, 0)

	jobCh <- job
	close(jobCh)
	wr.Wg.Wait()

	result := <-resultsCh
	if result.Success || !errors.Is(result.Err, apperrors.ErrUnknownOpType) {
		t.Fatalf("expected ErrUnknownOpType, got: %+v", result)
	}

	var pipelineErr *apperrors.JobPipelineError
	if !errors.As(result.Err, &pipelineErr) || pipelineErr.Step != apperrors.PipelineStepIntent {
		t.Fatalf("expected an intent-step pipeline error, got: %v", result.Err)
	}

	if _, err := registry.LoadJob(job.ID); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("a rejected intent must never claim an id, got: %v", err)
	}
}

func TestCurrencyMismatchFailsJobAndReleasesClaim(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t)))

	wrongCurrency := testIntent()
	wrongCurrency.CurrencyCode = "USD"
	job := NewWriteJob(wrongCurrency, 0)

	jobCh <- job
	close(jobCh)
	wr.Wg.Wait()

	result := <-resultsCh
	if result.Success || !errors.Is(result.Err, apperrors.ErrCurrencyMismatch) {
		t.Fatalf("expected ErrCurrencyMismatch, got: %+v", result)
	}

	var pipelineErr *apperrors.JobPipelineError
	if !errors.As(result.Err, &pipelineErr) || pipelineErr.Step != apperrors.PipelineStepCurrency {
		t.Fatalf("expected a currency-step pipeline error, got: %v", result.Err)
	}

	if _, err := registry.LoadJob(job.ID); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("a job rejected after registering must release its claim, got: %v", err)
	}
}

func TestReceiverCurrencyMismatchFailsJobAndReleasesClaim(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t)))

	toUSDAccount := testIntent()
	toUSDAccount.ToAccountID = "CUR-005"
	job := NewWriteJob(toUSDAccount, 0)

	jobCh <- job
	close(jobCh)
	wr.Wg.Wait()

	result := <-resultsCh
	if result.Success || !errors.Is(result.Err, apperrors.ErrCurrencyMismatch) {
		t.Fatalf("expected ErrCurrencyMismatch, got: %+v", result)
	}

	var mismatch *apperrors.CurrencyMismatchError
	if !errors.As(result.Err, &mismatch) || mismatch.AccountID != "CUR-005" {
		t.Fatalf("expected the mismatch to name the receiver, got: %v", result.Err)
	}

	var pipelineErr *apperrors.JobPipelineError
	if !errors.As(result.Err, &pipelineErr) || pipelineErr.Step != apperrors.PipelineStepCurrency {
		t.Fatalf("expected a currency-step pipeline error, got: %v", result.Err)
	}

	if _, err := registry.LoadJob(job.ID); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("a job rejected after registering must release its claim, got: %v", err)
	}
}

func TestUnsupportedOpTypeFailsJobAndReleasesClaim(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t)))

	house := testIntent()
	house.Type = OpHouseTransfer
	job := NewWriteJob(house, 0)

	jobCh <- job
	close(jobCh)
	wr.Wg.Wait()

	result := <-resultsCh
	if result.Success || !errors.Is(result.Err, apperrors.ErrOpTypeNotSupported) {
		t.Fatalf("expected ErrOpTypeNotSupported, got: %+v", result)
	}

	var pipelineErr *apperrors.JobPipelineError
	if !errors.As(result.Err, &pipelineErr) || pipelineErr.Step != apperrors.PipelineStepDispatch {
		t.Fatalf("expected a dispatch-step pipeline error, got: %v", result.Err)
	}

	if _, err := registry.LoadJob(job.ID); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("a job rejected after registering must release its claim, got: %v", err)
	}
}

func TestMissingSuspenseAccountFailsJobAndReleasesClaim(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, ledger.NewAccountStore()))

	job := NewWriteJob(testIntent(), 0)

	jobCh <- job
	close(jobCh)
	wr.Wg.Wait()

	result := <-resultsCh
	if result.Success {
		t.Fatalf("a store with no accounts must not let a job succeed, got: %+v", result)
	}
	if _, err := registry.LoadJob(job.ID); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("a failed job must release its claim, got: %v", err)
	}
}

func storeWithoutHouse(t *testing.T) *ledger.AccountStore {
	t.Helper()

	store := ledger.NewAccountStore()
	adds := []error{
		store.AddCurrent(&accounts.CurrentAccount{
			Account:  accounts.Account{BaseAccount: accounts.BaseAccount{AccountID: "CUR-001", CurrencyCode: "NGN"}},
			Customer: accounts.Customer{Tier: constants.TierTwo},
		}),
		store.AddCurrent(&accounts.CurrentAccount{
			Account:  accounts.Account{BaseAccount: accounts.BaseAccount{AccountID: "CUR-004", CurrencyCode: "NGN"}},
			Customer: accounts.Customer{Tier: constants.TierOne},
		}),
		store.AddSuspense(&accounts.Suspense{Account: accounts.Account{BaseAccount: accounts.BaseAccount{AccountID: "SUSPENSE-NGN", CurrencyCode: "NGN"}}}),
	}
	for _, err := range adds {
		if err != nil {
			t.Fatalf("seeding store: %v", err)
		}
	}
	return store
}

func TestMissingHouseAccountFailsWhenACutApplies(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, storeWithoutHouse(t)))

	job := NewWriteJob(testIntent(), 0)

	jobCh <- job
	close(jobCh)
	wr.Wg.Wait()

	result := <-resultsCh
	if result.Success || !errors.Is(result.Err, apperrors.ErrNoAccountForKind) {
		t.Fatalf("expected ErrNoAccountForKind, got: %+v", result)
	}

	var pipelineErr *apperrors.JobPipelineError
	if !errors.As(result.Err, &pipelineErr) || pipelineErr.Step != apperrors.PipelineStepBuildEntry {
		t.Fatalf("expected a build_entry-step pipeline error, got: %v", result.Err)
	}

	if _, err := registry.LoadJob(job.ID); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("a failed job must release its claim, got: %v", err)
	}
}

func TestMissingHouseAccountDoesNotMatterBelowTheThreshold(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, storeWithoutHouse(t)))

	small := testIntent()
	small.Amount = constants.HouseFeeThreshold
	job := NewWriteJob(small, 0)

	jobCh <- job
	close(jobCh)
	wr.Wg.Wait()

	result := <-resultsCh
	if !result.Success {
		t.Fatalf("no house cut applies at the threshold, so a missing house account must not matter, got: %v", result.Err)
	}
}

func TestWriteWorkRejectsWrongPayload(t *testing.T) {
	wr := NewWriteWorker(context.Background())
	work := wr.WriteWork(ledger.NewIDRegistry(), testStore(t))

	bad := &Job{ID: "bad", Payload: "not an intent", Timeout: time.Second}
	err := work(context.Background(), bad)
	if !errors.Is(err, apperrors.ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got: %v", err)
	}
}

func TestJobTimeoutFallsBackToDefault(t *testing.T) {
	job := &Job{}
	if job.timeout() != DefaultJobTimeout {
		t.Fatalf("zero timeout should fall back to default, got %s", job.timeout())
	}

	job.Timeout = time.Second
	if job.timeout() != time.Second {
		t.Fatalf("explicit timeout should be kept, got %s", job.timeout())
	}
}
