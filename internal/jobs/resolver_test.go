package jobs

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/journal"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

type recoverSpy struct {
	records      []journal.RawEntry
	recoverErr   error
	recoverCalls int
}

func (s *recoverSpy) Append([]byte) error { return nil }
func (s *recoverSpy) Close() error        { return nil }
func (s *recoverSpy) Reset()              {}
func (s *recoverSpy) Recover() ([]journal.RawEntry, error) {
	s.recoverCalls++
	return s.records, s.recoverErr
}

func spyWith(t *testing.T, ids ...string) *recoverSpy {
	t.Helper()
	spy := &recoverSpy{}
	for _, id := range ids {
		spy.records = append(spy.records, journal.RawEntry{Timestamp: 1, Payload: entryPayload(t, id)})
	}
	return spy
}

func pendingRegistry(t *testing.T, ids ...string) *ledger.IDRegistry {
	t.Helper()
	r := ledger.NewIDRegistry()
	for _, id := range ids {
		if _, err := r.Register(id); err != nil {
			t.Fatalf("Register(%s): %v", id, err)
		}
		if err := r.MarkPending(id); err != nil {
			t.Fatalf("MarkPending(%s): %v", id, err)
		}
	}
	return r
}

func fundedBalances() *ledger.Balances {
	b := ledger.NewBalances()
	b.Set("CUR-001", "NGN", 10000)
	b.Set("CUR-002", "NGN", 0)
	return b
}

func TestResolverDoesNotReadTheJournalWhenNothingIsPending(t *testing.T) {
	spy := spyWith(t, "job-1")

	result, err := StartResolverPipeline(context.Background(), spy, ledger.NewIDRegistry(), fundedBalances())
	if err != nil {
		t.Fatalf("StartResolverPipeline: %v", err)
	}
	if spy.recoverCalls != 0 || len(result.Completed) != 0 || len(result.Failed) != 0 {
		t.Fatalf("nothing pending means nothing to do: %+v, %d reads", result, spy.recoverCalls)
	}
}

func TestResolverCompletesAPendingJobFoundInTheJournalAndAppliesIt(t *testing.T) {
	registry := pendingRegistry(t, "job-1")
	balances := fundedBalances()

	result, err := StartResolverPipeline(context.Background(), spyWith(t, "job-1"), registry, balances)
	if err != nil {
		t.Fatalf("StartResolverPipeline: %v", err)
	}

	if !reflect.DeepEqual(result.Completed, []string{"job-1"}) || len(result.Failed) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if lookup, _ := registry.LoadJob("job-1"); lookup.State != ledger.StateCompleted {
		t.Fatalf("want completed, got %s", lookup.State)
	}
	if got := balances.Get("CUR-001", "NGN"); got != 9500 {
		t.Fatalf("the entry must be applied: want 9500, got %d", got)
	}
}

func TestResolverFailsAPendingJobThatNeverLandedAndFreesItsID(t *testing.T) {
	registry := pendingRegistry(t, "job-1")
	balances := fundedBalances()

	result, err := StartResolverPipeline(context.Background(), spyWith(t, "job-other"), registry, balances)
	if err != nil {
		t.Fatalf("StartResolverPipeline: %v", err)
	}

	if !reflect.DeepEqual(result.Failed, []string{"job-1"}) || len(result.Completed) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if res, err := registry.Register("job-1"); err != nil || !res.New {
		t.Fatalf("a job that never landed must be claimable again: %+v, %v", res, err)
	}
	if got := balances.Get("CUR-001", "NGN"); got != 10000 {
		t.Fatalf("nothing may be applied for a job that never landed, got %d", got)
	}
}

func TestResolverSplitsSeveralPendingJobsBetweenCompletedAndFailed(t *testing.T) {
	registry := pendingRegistry(t, "job-a", "job-b", "job-c")

	result, err := StartResolverPipeline(context.Background(), spyWith(t, "job-c", "job-a"), registry, fundedBalances())
	if err != nil {
		t.Fatalf("StartResolverPipeline: %v", err)
	}

	if !reflect.DeepEqual(result.Completed, []string{"job-c", "job-a"}) {
		t.Fatalf("completed follows journal order, got %v", result.Completed)
	}
	if !reflect.DeepEqual(result.Failed, []string{"job-b"}) {
		t.Fatalf("unexpected failed: %v", result.Failed)
	}
	if len(registry.PendingIDs()) != 0 {
		t.Fatal("nothing may stay pending")
	}
}

func TestResolverAppliesAPendingJobOnceWhenItsRecordRepeats(t *testing.T) {
	registry := pendingRegistry(t, "job-1")
	balances := fundedBalances()

	result, err := StartResolverPipeline(context.Background(), spyWith(t, "job-1", "job-1"), registry, balances)
	if err != nil {
		t.Fatalf("a repeated record must not abort the resolver: %v", err)
	}

	if len(result.Completed) != 1 {
		t.Fatalf("expected one completion, got %v", result.Completed)
	}
	if got := balances.Get("CUR-001", "NGN"); got != 9500 {
		t.Fatalf("applied twice? want 9500, got %d", got)
	}
}

func TestResolverSkipsCompletedAndUnknownIDsInTheJournal(t *testing.T) {
	registry := pendingRegistry(t, "job-pending")
	if _, err := registry.Register("job-done"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := registry.Complete("job-done"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	balances := fundedBalances()

	_, err := StartResolverPipeline(context.Background(), spyWith(t, "job-done", "job-unknown", "job-pending"), registry, balances)
	if err != nil {
		t.Fatalf("StartResolverPipeline: %v", err)
	}

	if got := balances.Get("CUR-001", "NGN"); got != 9500 {
		t.Fatalf("only the pending job may be applied: want 9500, got %d", got)
	}
}

func TestResolverRejectsAJobThatIsInFlightDuringResolution(t *testing.T) {
	registry := pendingRegistry(t, "job-pending")
	if _, err := registry.Register("job-running"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err := StartResolverPipeline(context.Background(), spyWith(t, "job-running"), registry, fundedBalances())

	if !errors.Is(err, apperrors.ErrUnexpectedJobState) {
		t.Fatalf("expected ErrUnexpectedJobState, got %v", err)
	}
	var resolveErr *apperrors.ResolvePipelineError
	if !errors.As(err, &resolveErr) || resolveErr.EntryID != "job-running" || resolveErr.Index != 0 {
		t.Fatalf("the error must name the entry and record, got %v", err)
	}
}

func TestResolverWrapsAFailedRecovery(t *testing.T) {
	spy := &recoverSpy{recoverErr: apperrors.ErrCorruptRecord}

	_, err := StartResolverPipeline(context.Background(), spy, pendingRegistry(t, "job-1"), fundedBalances())

	var resolveErr *apperrors.ResolvePipelineError
	if !errors.As(err, &resolveErr) || resolveErr.Step != apperrors.ResolveStepRecover {
		t.Fatalf("expected a recover-step error, got %v", err)
	}
	if !errors.Is(err, apperrors.ErrCorruptRecord) {
		t.Fatal("the cause must stay reachable")
	}
}

func TestResolverStopsAtAnUndecodableRecord(t *testing.T) {
	spy := spyWith(t, "job-1")
	spy.records = append([]journal.RawEntry{{Timestamp: 1, Payload: []byte("not json")}}, spy.records...)

	_, err := StartResolverPipeline(context.Background(), spy, pendingRegistry(t, "job-1"), fundedBalances())

	var resolveErr *apperrors.ResolvePipelineError
	if !errors.As(err, &resolveErr) || resolveErr.Step != apperrors.ResolveStepDecode || resolveErr.Index != 0 {
		t.Fatalf("expected a decode-step error at record 0, got %v", err)
	}
}

func TestResolverLeavesTheJobPendingWhenApplyFails(t *testing.T) {
	registry := pendingRegistry(t, "job-1")
	balances := ledger.NewBalances()
	balances.Set("CUR-002", "NGN", math.MaxInt64)

	_, err := StartResolverPipeline(context.Background(), spyWith(t, "job-1"), registry, balances)

	var resolveErr *apperrors.ResolvePipelineError
	if !errors.As(err, &resolveErr) || resolveErr.Step != apperrors.ResolveStepApply {
		t.Fatalf("expected an apply-step error, got %v", err)
	}
	if lookup, _ := registry.LoadJob("job-1"); lookup.State != ledger.StatePending {
		t.Fatalf("a job whose entry was not applied must stay pending, got %s", lookup.State)
	}
}

func TestResolverStopsOnACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	spy := spyWith(t, "job-1")

	_, err := StartResolverPipeline(ctx, spy, pendingRegistry(t, "job-1"), fundedBalances())

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if spy.recoverCalls != 0 {
		t.Fatal("a cancelled resolver must not read the journal")
	}
}
