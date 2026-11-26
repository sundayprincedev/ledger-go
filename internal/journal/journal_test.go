package journal

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func TestAppendAndRecover(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.log")

	lw, err := NewLogWriter(path)
	if err != nil {
		t.Fatalf("NewLogWriter: %v", err)
	}

	payloads := [][]byte{[]byte("first"), []byte("second"), []byte("third")}
	for _, p := range payloads {
		if err := lw.Append(p); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := lw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := RecoverAndClean(path)
	if err != nil {
		t.Fatalf("RecoverAndClean: %v", err)
	}
	if len(entries) != len(payloads) {
		t.Fatalf("expected %d entries, got %d", len(payloads), len(entries))
	}
	for i, e := range entries {
		if !bytes.Equal(e.Payload, payloads[i]) {
			t.Fatalf("entry %d payload mismatch: got %q want %q", i, e.Payload, payloads[i])
		}
	}
}

func TestAppendAfterPoisonFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.log")

	lw, err := NewLogWriter(path)
	if err != nil {
		t.Fatalf("NewLogWriter: %v", err)
	}
	defer lw.Close()

	lw.poisoned = true
	if err := lw.Append([]byte("x")); err == nil {
		t.Fatal("expected error on poisoned writer")
	}
}

func TestRecoverTruncatesTornTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.log")

	lw, err := NewLogWriter(path)
	if err != nil {
		t.Fatalf("NewLogWriter: %v", err)
	}
	if err := lw.Append([]byte("whole")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := lw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open for torn append: %v", err)
	}
	if _, err := f.Write([]byte{0x01, 0x02, 0x03}); err != nil {
		t.Fatalf("write torn tail: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close torn append: %v", err)
	}

	entries, err := RecoverAndClean(path)
	if err != nil {
		t.Fatalf("RecoverAndClean: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestRecoverAndCleanTreatsAMissingFileAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "never-written.log")

	entries, err := RecoverAndClean(path)
	if err != nil {
		t.Fatalf("a journal that does not exist yet is an empty journal, got: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("recovery must not create the file (err: %v)", err)
	}
}

func TestResetLetsAPoisonedJournalAppendAgain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.log")
	j := Open(path)
	defer j.Close()

	if err := j.Append([]byte("first")); err != nil {
		t.Fatalf("Append: %v", err)
	}

	_ = j.writer.file.Close()
	if err := j.Append([]byte("lost")); err == nil {
		t.Fatal("writing to a broken file must fail")
	}
	if err := j.Append([]byte("blocked")); !errors.Is(err, apperrors.ErrWriterPoisoned) {
		t.Fatalf("the writer must be poisoned, got %v", err)
	}

	j.Reset()

	if err := j.Append([]byte("second")); err != nil {
		t.Fatalf("a reset journal must accept appends: %v", err)
	}

	entries, err := j.Recover()
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(entries) != 2 || string(entries[0].Payload) != "first" || string(entries[1].Payload) != "second" {
		t.Fatalf("expected first and second only, got %d entries", len(entries))
	}
}

func TestResetOnAJournalThatNeverWroteIsANoOp(t *testing.T) {
	j := Open(filepath.Join(t.TempDir(), "ledger.log"))
	j.Reset()
	if err := j.Append([]byte("x")); err != nil {
		t.Fatalf("Append after a no-op reset: %v", err)
	}
	_ = j.Close()
}

func TestPoisonBlocksAppendsUntilTheNextReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.log")
	j := Open(path)
	defer j.Close()

	if err := j.Append([]byte("first")); err != nil {
		t.Fatalf("Append: %v", err)
	}

	j.Poison()
	if err := j.Append([]byte("blocked")); !errors.Is(err, apperrors.ErrWriterPoisoned) {
		t.Fatalf("a poisoned journal must refuse appends, got %v", err)
	}
	if err := j.Append([]byte("still blocked")); !errors.Is(err, apperrors.ErrWriterPoisoned) {
		t.Fatalf("poison must hold across attempts, got %v", err)
	}

	j.Reset()
	if err := j.Append([]byte("second")); err != nil {
		t.Fatalf("Reset must lift the poison: %v", err)
	}

	entries, err := j.Recover()
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected first and second only, got %d", len(entries))
	}
}
