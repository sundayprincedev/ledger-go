package main

import (
	"errors"
	"testing"

	"github.com/sundayprincedev/ledger/internal/accounts"
	"github.com/sundayprincedev/ledger/internal/constants"
	"github.com/sundayprincedev/ledger/internal/data"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

const seedFile = "../../internal/data/seed.json"

func TestBuildStoreFromSeed(t *testing.T) {
	seed, err := data.Load(seedFile)
	if err != nil {
		t.Fatalf("loading the seed: %v", err)
	}

	store, err := buildStore(seed.Accounts)
	if err != nil {
		t.Fatalf("building the store: %v", err)
	}

	for _, sa := range seed.Accounts {
		profile, err := store.Lookup(sa.AccountID)
		if err != nil {
			t.Fatalf("account %s missing from the store: %v", sa.AccountID, err)
		}
		if profile.Kind != accounts.Kind(sa.Kind) || profile.CurrencyCode != sa.CurrencyCode {
			t.Fatalf("account %s: want %s/%s, got %s/%s", sa.AccountID, sa.Kind, sa.CurrencyCode, profile.Kind, profile.CurrencyCode)
		}

		wantTier := constants.KYCTier(sa.Tier)
		if profile.Tier != wantTier {
			t.Fatalf("account %s: want tier %d, got %d", sa.AccountID, wantTier, profile.Tier)
		}
	}

	for _, currency := range []string{"NGN", "USD", "GBP"} {
		house, err := store.FindByKind(accounts.KindHouse, currency)
		if err != nil || house.ID != "HOUSE-"+currency {
			t.Fatalf("house account for %s: got %+v (err: %v)", currency, house, err)
		}
		suspense, err := store.FindByKind(accounts.KindSuspense, currency)
		if err != nil || suspense.ID != "SUSPENSE-"+currency {
			t.Fatalf("suspense account for %s: got %+v (err: %v)", currency, suspense, err)
		}
	}
}

func TestBuildStoreRejectsAnUnknownKind(t *testing.T) {
	_, err := buildStore([]data.SeedAccount{{AccountID: "X-1", Kind: "vault", CurrencyCode: "NGN"}})

	if !errors.Is(err, apperrors.ErrUnknownAccountKind) {
		t.Fatalf("expected ErrUnknownAccountKind, got: %v", err)
	}

	var kindErr *apperrors.KindError
	if !errors.As(err, &kindErr) || kindErr.Kind != "vault" {
		t.Fatalf("expected the error to carry the offending kind, got: %v", err)
	}
}

func TestBuildStoreRejectsDuplicateAccounts(t *testing.T) {
	_, err := buildStore([]data.SeedAccount{
		{AccountID: "CUR-001", Kind: "customer_current", CurrencyCode: "NGN", Tier: 1},
		{AccountID: "CUR-001", Kind: "customer_current", CurrencyCode: "NGN", Tier: 1},
	})

	if !errors.Is(err, apperrors.ErrAccountExists) {
		t.Fatalf("expected ErrAccountExists, got: %v", err)
	}
}
