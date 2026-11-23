package jobs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/journal"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

func readTestEntry(id string) ledger.Entry {
	return ledger.Entry{
		ID: id,
		Lines: []ledger.Line{
			{AccountID: "CUR-001", Direction: ledger.Debit, Amount: 500, CurrencyCode: "NGN", Reference: ledger.Reference{JobID: id, CounterpartyAccountID: "CUR-002"}},
			{AccountID: "CUR-002", Direction: ledger.Credit, Amount: 500, CurrencyCode: "NGN", Reference: ledger.Reference{JobID: id, CounterpartyAccountID: "CUR-001"}},
		},
	}
}

func writeJournal(t *testing.T, payloads ...[]byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "ledger.log")
	lw, err := journal.NewLogWriter(path)
	if err != nil {
		t.Fatalf("NewLogWriter: %v", err)
	}
	for _, p := range payloads {
		if err := lw.Append(p); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := lw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return path
}

func entryPayload(t *testing.T, id string) []byte {
	t.Helper()

	e := readTestEntry(id)
	payload, err := e.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return payload
}

func TestReadPipelineRegistersEveryRecoveredEntryAsCompleted(t *testing.T) {
	path := writeJournal(t, entryPayload(t, "job-1"), entryPayload(t, "job-2"), entryPayload(t, "job-3"))
	registry := ledger.NewIDRegistry()

	result, err := StartReadPipeline(context.Background(), path, registry)
	if err != nil {
		t.Fatalf("StartReadPipeline: %v", err)
	}

	if result.Recovered != 3 || result.Registered != 3 || len(result.Duplicates) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	for _, id := range []string{"job-1", "job-2", "job-3"} {
		lookup, err := registry.LoadJob(id)
		if err != nil {
			t.Fatalf("%s must be in the registry: %v", id, err)
		}
		if lookup.State != ledger.StateCompleted {
			t.Fatalf("%s should be completed, got %s", id, lookup.State)
		}
	}
}

func TestReadPipelineMakesReplayedJobsRejectedAsCompletedDuplicates(t *testing.T) {
	path := writeJournal(t, entryPayload(t, "job-1"))
	registry := ledger.NewIDRegistry()

	if _, err := StartReadPipeline(context.Background(), path, registry); err != nil {
		t.Fatalf("StartReadPipeline: %v", err)
	}

	again, err := registry.Register("job-1")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if again.New || again.State != ledger.StateCompleted {
		t.Fatalf("a replayed job must not be claimable again: %+v", again)
	}
}

func TestReadPipelineReportsDuplicateRecordsAndKeepsGoing(t *testing.T) {
	path := writeJournal(t, entryPayload(t, "job-1"), entryPayload(t, "job-2"), entryPayload(t, "job-1"), entryPayload(t, "job-3"))
	registry := ledger.NewIDRegistry()

	result, err := StartReadPipeline(context.Background(), path, registry)
	if err != nil {
		t.Fatalf("a duplicate record must not stop replay: %v", err)
	}

	if result.Recovered != 4 || result.Registered != 3 || len(result.Duplicates) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Recovered != result.Registered+len(result.Duplicates) {
		t.Fatalf("recovered must equal registered plus duplicates: %+v", result)
	}
	dup := result.Duplicates[0]
	if dup.EntryID != "job-1" || dup.Index != 2 {
		t.Fatalf("unexpected duplicate report: %+v", dup)
	}
	if !errors.Is(dup, apperrors.ErrDuplicateJournalEntry) {
		t.Fatal("the duplicate report must unwrap to ErrDuplicateJournalEntry")
	}
	if _, err := registry.LoadJob("job-3"); err != nil {
		t.Fatalf("records after a duplicate must still be registered: %v", err)
	}
}

func TestReadPipelineStopsAtTheFirstUndecodableRecord(t *testing.T) {
	unbalanced := readTestEntry("job-bad")
	unbalanced.Lines[1].Amount = 400
	badPayload, err := unbalanced.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	path := writeJournal(t, entryPayload(t, "job-1"), badPayload, entryPayload(t, "job-3"))
	registry := ledger.NewIDRegistry()

	result, err := StartReadPipeline(context.Background(), path, registry)

	var readErr *apperrors.ReadPipelineError
	if !errors.As(err, &readErr) {
		t.Fatalf("expected a ReadPipelineError, got %T: %v", err, err)
	}
	if readErr.Step != apperrors.ReadStepDecode || readErr.Index != 1 {
		t.Fatalf("expected a decode failure at record 1, got %+v", readErr)
	}
	var balanceErr *apperrors.BalanceError
	if !errors.As(err, &balanceErr) {
		t.Fatalf("the cause must be the balance error, got %v", err)
	}
	if result.Registered != 1 {
		t.Fatalf("only the record before the bad one is registered, got %d", result.Registered)
	}
	if _, err := registry.LoadJob("job-3"); err == nil {
		t.Fatal("nothing after the bad record may be registered")
	}
}

func TestReadPipelineRejectsGarbageThatPassesTheCRC(t *testing.T) {
	path := writeJournal(t, []byte("not json"))

	_, err := StartReadPipeline(context.Background(), path, ledger.NewIDRegistry())

	var decodeErr *apperrors.EntryDecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("expected an EntryDecodeError, got %v", err)
	}
}

func TestReadPipelineTreatsAMissingJournalAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "never-written.log")

	result, err := StartReadPipeline(context.Background(), path, ledger.NewIDRegistry())
	if err != nil {
		t.Fatalf("a journal that does not exist yet is not an error: %v", err)
	}
	if result.Recovered != 0 || result.Registered != 0 || len(result.Duplicates) != 0 {
		t.Fatalf("expected an empty result, got %+v", result)
	}
}

func TestReadPipelineTruncatesATornTailBeforeReplay(t *testing.T) {
	path := writeJournal(t, entryPayload(t, "job-1"), entryPayload(t, "job-2"))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if err := os.Truncate(path, info.Size()-3); err != nil {
		t.Fatalf("Truncate: %v", err)
	}

	registry := ledger.NewIDRegistry()
	result, err := StartReadPipeline(context.Background(), path, registry)
	if err != nil {
		t.Fatalf("a torn tail is cleaned, not fatal: %v", err)
	}

	if result.Recovered != 1 || result.Registered != 1 {
		t.Fatalf("only the intact record should replay: %+v", result)
	}
	if _, err := registry.LoadJob("job-2"); err == nil {
		t.Fatal("the torn record must not be registered")
	}
}

func TestReadPipelineStopsOnACancelledContext(t *testing.T) {
	path := writeJournal(t, entryPayload(t, "job-1"))
	registry := ledger.NewIDRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := StartReadPipeline(ctx, path, registry)

	var readErr *apperrors.ReadPipelineError
	if !errors.As(err, &readErr) || readErr.Step != apperrors.ReadStepContext {
		t.Fatalf("expected a context-step ReadPipelineError, got %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatal("the cause must be context.Canceled")
	}
	if _, err := registry.LoadJob("job-1"); err == nil {
		t.Fatal("nothing may be registered after cancellation")
	}
}
