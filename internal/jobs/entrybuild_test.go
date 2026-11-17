package jobs

import (
	"testing"

	"github.com/sundayprincedev/ledger/internal/constants"
	"github.com/sundayprincedev/ledger/internal/ledger"
	"github.com/sundayprincedev/ledger/internal/money"
)

func netEffects(lines []ledger.Line) map[string]int64 {
	net := make(map[string]int64)
	for _, line := range lines {
		if line.Direction == ledger.Credit {
			net[line.AccountID] += line.Amount
		} else {
			net[line.AccountID] -= line.Amount
		}
	}
	return net
}

func line(account string, direction ledger.Direction, amount int64, counterparty string) ledger.Line {
	return ledger.Line{
		AccountID:    account,
		Direction:    direction,
		Amount:       amount,
		CurrencyCode: "NGN",
		Reference:    ledger.Reference{JobID: "job-1", CounterpartyAccountID: counterparty},
	}
}

func assertLines(t *testing.T, got, want []ledger.Line) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d lines, got %d: %+v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d:\nwant %+v\ngot  %+v", i, want[i], got[i])
		}
	}
}

func TestBuildDirectEntryWithHouseCut(t *testing.T) {
	intent := testIntent()
	shares, err := money.SplitHouseCut(intent.Amount)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	house, receiver, total := shares[0], shares[1], shares[0]+shares[1]
	if house <= 0 {
		t.Fatalf("this test needs a transfer above the threshold, got house share %d", house)
	}

	entry, err := buildDirectEntry(testStore(t), "job-1", intent, shares)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if entry.ID != "job-1" {
		t.Fatalf("entry id should be the job id, got %q", entry.ID)
	}

	assertLines(t, entry.Lines, []ledger.Line{
		line("CUR-001", ledger.Debit, total, "SUSPENSE-NGN"),
		line("SUSPENSE-NGN", ledger.Credit, total, "CUR-001"),
		line("HOUSE-NGN", ledger.Credit, house, "SUSPENSE-NGN"),
		line("SUSPENSE-NGN", ledger.Debit, house, "HOUSE-NGN"),
		line("CUR-004", ledger.Credit, receiver, "SUSPENSE-NGN"),
		line("SUSPENSE-NGN", ledger.Debit, receiver, "CUR-004"),
	})

	if err := entry.Validate(); err != nil {
		t.Fatalf("a built entry must validate: %v", err)
	}

	net := netEffects(entry.Lines)
	if net["CUR-001"] != -intent.Amount {
		t.Fatalf("sender should lose exactly the amount, net %d", net["CUR-001"])
	}
	if net["HOUSE-NGN"] != house {
		t.Fatalf("house should gain its share %d, net %d", house, net["HOUSE-NGN"])
	}
	if net["CUR-004"] != receiver {
		t.Fatalf("receiver should gain its share %d, net %d", receiver, net["CUR-004"])
	}
	if net["SUSPENSE-NGN"] != 0 {
		t.Fatalf("suspense must end at zero, net %d", net["SUSPENSE-NGN"])
	}
}

func TestBuildDirectEntryWithoutHouseCut(t *testing.T) {
	intent := testIntent()
	intent.Amount = constants.HouseFeeThreshold
	shares, err := money.SplitHouseCut(intent.Amount)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if shares[0] != 0 {
		t.Fatalf("this test needs a transfer with no cut, got house share %d", shares[0])
	}

	entry, err := buildDirectEntry(testStore(t), "job-1", intent, shares)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	assertLines(t, entry.Lines, []ledger.Line{
		line("CUR-001", ledger.Debit, intent.Amount, "SUSPENSE-NGN"),
		line("SUSPENSE-NGN", ledger.Credit, intent.Amount, "CUR-001"),
		line("CUR-004", ledger.Credit, intent.Amount, "SUSPENSE-NGN"),
		line("SUSPENSE-NGN", ledger.Debit, intent.Amount, "CUR-004"),
	})

	if err := entry.Validate(); err != nil {
		t.Fatalf("a built entry must validate: %v", err)
	}

	net := netEffects(entry.Lines)
	if _, touched := net["HOUSE-NGN"]; touched {
		t.Fatal("a transfer with no cut must not touch the house account")
	}
	if net["CUR-001"] != -intent.Amount || net["CUR-004"] != intent.Amount || net["SUSPENSE-NGN"] != 0 {
		t.Fatalf("unexpected net effects: %+v", net)
	}
}
