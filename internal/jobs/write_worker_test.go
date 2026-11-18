package jobs

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/sundayprincedev/ledger/internal/accounts"
	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/journal"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

func testWriter(t *testing.T) ledger.Journal {
	t.Helper()

	file := journal.Open(filepath.Join(t.TempDir(), "ledger.log"))
	t.Cleanup(func() { _ = file.Close() })
	return file
}

type fakeJournal struct {
	appendErr   error
	appendCalls int
}

func (f *fakeJournal) Append(payload []byte) error {
	f.appendCalls++
	return f.appendErr
}

func (f *fakeJournal) Recover() ([]journal.RawEntry, error) {
	return nil, nil
}

func (f *fakeJournal) Close() error {
	return nil
}

func runJobs(t *testing.T, registry *ledger.IDRegistry, target ledger.Journal, batch ...*Job) []JobResult {
	t.Helper()

	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, len(batch))
	errCh := make(chan error, 2*len(batch))
	resultsCh := make(chan JobResult, len(batch))

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t), target))

	for _, job := range batch {
		jobCh <- job
	}
	close(jobCh)
	wr.Wg.Wait()

	results := make([]JobResult, 0, len(batch))
	for range batch {
		results = append(results, <-resultsCh)
	}
	return results
}

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

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t), testWriter(t)))

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

func TestSuccessfulJobAppendsItsEntryToTheJournal(t *testing.T) {
	registry := ledger.NewIDRegistry()
	target := testWriter(t)
	job := NewWriteJob(testIntent(), 0)

	if result := runJobs(t, registry, target, job)[0]; !result.Success {
		t.Fatalf("expected success, got: %v", result.Err)
	}

	records, err := target.Recover()
	if err != nil {
		t.Fatalf("recovering the journal: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected exactly one journal record, got %d", len(records))
	}

	entry, err := ledger.EntryFromPayload(records[0].Payload, time.Unix(0, records[0].Timestamp))
	if err != nil {
		t.Fatalf("the appended payload must decode and validate: %v", err)
	}
	if entry.ID != job.ID || len(entry.Lines) != 6 {
		t.Fatalf("expected entry %s with 6 lines, got %s with %d", job.ID, entry.ID, len(entry.Lines))
	}

	if state, err := registry.LoadJob(job.ID); err != nil || state.State != ledger.StateCompleted {
		t.Fatalf("a committed job must be completed in the registry, got %+v (err: %v)", state, err)
	}
}

func TestAppendFailureClasses(t *testing.T) {
	diskFull := errors.New("disk full")

	tests := []struct {
		name          string
		appendErr     error
		wantRecovery  bool
		wantUnknown   bool
		wantClaimKept bool
		wantState     JobState
	}{
		{"write failed", &apperrors.WritePipelineError{Step: apperrors.StepWriteRecord, Err: diskFull}, true, false, false, StateFailed},
		{"sync failed", &apperrors.WritePipelineError{Step: apperrors.StepSyncRecord, Err: diskFull}, true, true, true, StateOutcomePending},
		{"writer already poisoned", apperrors.ErrWriterPoisoned, true, false, false, StateFailed},
		{"payload too large", apperrors.ErrPayloadTooLarge, false, false, false, StateFailed},
		{"journal file could not be opened", &apperrors.WritePipelineError{Step: apperrors.StepOpenFile, Err: diskFull}, false, false, false, StateFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := ledger.NewIDRegistry()
			job := NewWriteJob(testIntent(), 0)

			result := runJobs(t, registry, &fakeJournal{appendErr: tc.appendErr}, job)[0]
			if result.Success {
				t.Fatal("a job whose append failed must not succeed")
			}

			var pipelineErr *apperrors.JobPipelineError
			if !errors.As(result.Err, &pipelineErr) || pipelineErr.Step != apperrors.PipelineStepAppend {
				t.Fatalf("expected an append-step pipeline error, got: %v", result.Err)
			}
			if !errors.Is(result.Err, tc.appendErr) {
				t.Fatalf("the journal's own error must stay reachable, got: %v", result.Err)
			}
			if got := errors.Is(result.Err, apperrors.ErrRecoveryRequired); got != tc.wantRecovery {
				t.Fatalf("recovery required: want %v, got %v", tc.wantRecovery, got)
			}
			if got := errors.Is(result.Err, apperrors.ErrOutcomeUnknown); got != tc.wantUnknown {
				t.Fatalf("outcome unknown: want %v, got %v", tc.wantUnknown, got)
			}
			if got := JobState(job.State.Load()); got != tc.wantState {
				t.Fatalf("job state: want %s, got %s", tc.wantState, got)
			}

			state, err := registry.LoadJob(job.ID)
			if tc.wantClaimKept {
				if err != nil || state.State != ledger.StateInFlight {
					t.Fatalf("an unknown outcome must keep the claim in flight, got %+v (err: %v)", state, err)
				}
				return
			}
			if !errors.Is(err, apperrors.ErrJobNotFound) {
				t.Fatalf("nothing was committed, so the claim must be released, got: %v", err)
			}
		})
	}
}

func TestRetryAfterAnUnknownOutcomeIsNotAppendedAgain(t *testing.T) {
	registry := ledger.NewIDRegistry()
	fake := &fakeJournal{appendErr: &apperrors.WritePipelineError{Step: apperrors.StepSyncRecord, Err: errors.New("fsync failed")}}

	results := runJobs(t, registry, fake, NewWriteJob(testIntent(), 0), NewWriteJob(testIntent(), 0))

	if !errors.Is(results[0].Err, apperrors.ErrOutcomeUnknown) {
		t.Fatalf("the first job should report an unknown outcome, got: %v", results[0].Err)
	}
	if !errors.Is(results[1].Err, apperrors.ErrJobAlreadyInFlight) {
		t.Fatalf("the retry must be held back while the first outcome is unknown, got: %v", results[1].Err)
	}
	if fake.appendCalls != 1 {
		t.Fatalf("the retry must never reach the journal, got %d appends", fake.appendCalls)
	}
}

func TestInvalidIntentIsRejectedBeforeRegister(t *testing.T) {
	registry := ledger.NewIDRegistry()
	wr := NewWriteWorker(context.Background())

	jobCh := make(chan *Job, 1)
	errCh := make(chan error, 2)
	resultsCh := make(chan JobResult, 1)

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t), testWriter(t)))

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

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t), testWriter(t)))

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

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t), testWriter(t)))

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

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, testStore(t), testWriter(t)))

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

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, ledger.NewAccountStore(), testWriter(t)))

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

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, storeWithoutHouse(t), testWriter(t)))

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

	wr.Start(1, jobCh, errCh, resultsCh, wr.WriteWork(registry, storeWithoutHouse(t), testWriter(t)))

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
	work := wr.WriteWork(ledger.NewIDRegistry(), testStore(t), testWriter(t))

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
