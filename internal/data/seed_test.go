package data

import "testing"

func TestLoadSeedData(t *testing.T) {
	seed, err := Load("seed.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(seed.Accounts) == 0 {
		t.Fatal("expected seed accounts")
	}

	if len(seed.Operations) == 0 {
		t.Fatal("expected seed operations")
	}
}
