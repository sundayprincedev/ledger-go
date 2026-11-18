package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sundayprincedev/ledger/internal/journal"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

func testConfig(t *testing.T) config {
	t.Helper()

	return config{
		seedPath:    seedFile,
		journalPath: filepath.Join(t.TempDir(), "ledger.log"),
		grace:       time.Second,
	}
}

func TestRunEndToEnd(t *testing.T) {
	cfg := testConfig(t)
	var out bytes.Buffer

	result, err := run(context.Background(), cfg, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if result.sent != 7 || result.tally.succeeded != 5 || result.tally.failed != 2 || result.tally.outcomePending != 0 {
		t.Fatalf("unexpected result: %+v\n%s", result, out.String())
	}
	if result.forced {
		t.Fatal("nothing should have needed the grace period")
	}

	log := out.String()
	for _, want := range []string{"transfer limit", "not supported yet"} {
		if !strings.Contains(log, want) {
			t.Fatalf("expected the output to mention %q:\n%s", want, log)
		}
	}

	file := journal.Open(cfg.journalPath)
	defer file.Close()

	records, err := file.Recover()
	if err != nil {
		t.Fatalf("recovering the journal: %v", err)
	}
	if len(records) != 5 {
		t.Fatalf("expected one journal record per successful job, got %d", len(records))
	}

	seen := make(map[string]bool)
	for _, record := range records {
		entry, err := ledger.EntryFromPayload(record.Payload, time.Unix(0, record.Timestamp))
		if err != nil {
			t.Fatalf("a journal record must decode and validate: %v", err)
		}
		if seen[entry.ID] {
			t.Fatalf("entry %s was written twice", entry.ID)
		}
		seen[entry.ID] = true
	}
}

func TestRunStopsFeedingOnceTheContextIsCancelled(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var out bytes.Buffer
	result, err := run(ctx, cfg, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if result.sent != 0 {
		t.Fatalf("no job should be sent after a signal, got %d", result.sent)
	}
	if _, err := os.Stat(cfg.journalPath); !os.IsNotExist(err) {
		t.Fatalf("nothing was appended, so the journal file should not exist (err: %v)", err)
	}
}

func TestRunReportsAMissingSeedFile(t *testing.T) {
	cfg := testConfig(t)
	cfg.seedPath = filepath.Join(t.TempDir(), "missing.json")

	if _, err := run(context.Background(), cfg, &bytes.Buffer{}); err == nil {
		t.Fatal("a missing seed file must fail the run")
	}
}

func TestDrainReturnsWithoutAnySignal(t *testing.T) {
	done := make(chan struct{})
	close(done)

	var cancelled atomic.Bool
	forced := drain(context.Background(), done, time.Second, func() { cancelled.Store(true) })

	if forced || cancelled.Load() {
		t.Fatal("work that finished on its own must not be cancelled")
	}
}

func TestDrainLetsWorkFinishInsideTheGracePeriod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	var cancelled atomic.Bool
	result := make(chan bool, 1)
	go func() {
		result <- drain(ctx, done, 2*time.Second, func() { cancelled.Store(true) })
	}()

	cancel()
	time.Sleep(20 * time.Millisecond)
	close(done)

	if forced := <-result; forced || cancelled.Load() {
		t.Fatal("work that finishes inside the grace period must not be cancelled")
	}
}

func TestDrainCancelsWorkAfterTheGracePeriod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	cancelled := make(chan struct{})

	result := make(chan bool, 1)
	go func() {
		result <- drain(ctx, done, 50*time.Millisecond, func() { close(cancelled) })
	}()

	cancel()

	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("work was not cancelled once the grace period expired")
	}
	close(done)

	if forced := <-result; !forced {
		t.Fatal("expected the drain to report that it had to force cancellation")
	}
}
