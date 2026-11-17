package ledger

import (
	"errors"
	"slices"
	"testing"
	"time"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func payloadEntry() Entry {
	return Entry{
		ID: "job-1",
		Lines: []Line{
			{AccountID: "CUR-001", Direction: Debit, Amount: 500, CurrencyCode: "NGN", Reference: Reference{JobID: "job-1", CounterpartyAccountID: "SUSPENSE-NGN"}},
			{AccountID: "SUSPENSE-NGN", Direction: Credit, Amount: 500, CurrencyCode: "NGN", Reference: Reference{JobID: "job-1", CounterpartyAccountID: "CUR-001"}},
		},
	}
}

func TestEntryPayloadRoundTrip(t *testing.T) {
	entry := payloadEntry()

	payload, err := entry.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	decoded, err := EntryFromPayload(payload, time.Now())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.ID != entry.ID || !slices.Equal(decoded.Lines, entry.Lines) {
		t.Fatalf("payload did not round trip:\nwant %+v\ngot  %+v", entry, decoded)
	}
}

func TestEntryFromPayloadRejectsGarbage(t *testing.T) {
	_, err := EntryFromPayload([]byte("not json"), time.Now())

	var decodeErr *apperrors.EntryDecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("expected an EntryDecodeError, got: %v", err)
	}
}

func TestEntryFromPayloadStillValidates(t *testing.T) {
	unbalanced := payloadEntry()
	unbalanced.Lines[1].Amount = 400

	payload, err := unbalanced.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if _, err := EntryFromPayload(payload, time.Now()); !errors.Is(err, apperrors.ErrUnbalancedEntry) {
		t.Fatalf("a decoded payload must be validated, expected ErrUnbalancedEntry, got: %v", err)
	}
}
