package journal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
