package main

import (
	"testing"

	"github.com/sundayprincedev/ledger/internal/data"
	"github.com/sundayprincedev/ledger/internal/jobs"
)

func TestBuildIntentsFromSeed(t *testing.T) {
	seed, err := data.Load(seedFile)
	if err != nil {
		t.Fatalf("loading the seed: %v", err)
	}

	intents, skipped := buildIntents(seed.Operations)

	counts := make(map[jobs.OpType]int)
	for _, intent := range intents {
		counts[intent.Type]++
	}
	if counts[jobs.OpSingleTransfer] != 6 || counts[jobs.OpBulkTransfer] != 1 || len(intents) != 7 {
		t.Fatalf("expected 6 single and 1 bulk intents, got %v", counts)
	}

	wantSkipped := map[string]int{"deposit": 2, "merchant_payout": 1, "fee": 1, "interest_accrual": 1}
	if len(skipped) != len(wantSkipped) {
		t.Fatalf("expected skipped %v, got %v", wantSkipped, skipped)
	}
	for opType, want := range wantSkipped {
		if skipped[opType] != want {
			t.Fatalf("expected %d skipped %s operations, got %d", want, opType, skipped[opType])
		}
	}

	for _, intent := range intents {
		if err := intent.Validate(); err != nil {
			t.Fatalf("a seeded intent must pass validation: %+v: %v", intent, err)
		}
	}
}

func TestBuildIntentsMapsTheBulkRecipients(t *testing.T) {
	intents, _ := buildIntents([]data.SeedOperation{{
		Type:          "bulk_transfer",
		FromAccountID: "CUR-001",
		CurrencyCode:  "NGN",
		Amount:        100001,
		Recipients:    []data.SeedRecipient{{AccountID: "SAV-001", Weight: 500}, {AccountID: "CUR-004", Weight: 500}},
	}})

	if len(intents) != 1 {
		t.Fatalf("expected one intent, got %d", len(intents))
	}
	got := intents[0]
	if got.Type != jobs.OpBulkTransfer || got.FromAccountID != "CUR-001" || got.Amount != 100001 {
		t.Fatalf("unexpected intent: %+v", got)
	}
	if len(got.Recipients) != 2 || got.Recipients[0] != (jobs.Recipient{AccountID: "SAV-001", Weight: 500}) {
		t.Fatalf("unexpected recipients: %+v", got.Recipients)
	}
}
