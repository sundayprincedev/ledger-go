package ledger

import (
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/sundayprincedev/ledger/internal/accounts"
	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func base(id, currency string) accounts.Account {
	return accounts.Account{BaseAccount: accounts.BaseAccount{AccountID: id, CurrencyCode: currency}}
}

func TestAccountStoreRecordsKindCurrencyAndTier(t *testing.T) {
	store := NewAccountStore()

	adds := []error{
		store.AddCurrent(&accounts.CurrentAccount{
			Account:  base("CUR-001", "NGN"),
			Customer: accounts.Customer{CustomerID: "CUST-001", Tier: constants.TierTwo},
		}),
		store.AddSavings(&accounts.SavingsAccount{
			Account:  base("SAV-002", "USD"),
			Customer: accounts.Customer{CustomerID: "CUST-002", Tier: constants.TierThree},
		}),
		store.AddMerchant(&accounts.Merchant{Account: base("MERCH-001", "NGN"), Tier: constants.TierOne}),
		store.AddHouse(&accounts.House{Account: base("HOUSE-NGN", "NGN")}),
		store.AddSuspense(&accounts.Suspense{Account: base("SUSPENSE-GBP", "GBP")}),
	}
	for i, err := range adds {
		if err != nil {
			t.Fatalf("add %d failed: %v", i, err)
		}
	}

	want := []AccountProfile{
		{ID: "CUR-001", Kind: accounts.KindCustomerCurrent, CurrencyCode: "NGN", Tier: constants.TierTwo},
		{ID: "SAV-002", Kind: accounts.KindCustomerSavings, CurrencyCode: "USD", Tier: constants.TierThree},
		{ID: "MERCH-001", Kind: accounts.KindMerchant, CurrencyCode: "NGN", Tier: constants.TierOne},
		{ID: "HOUSE-NGN", Kind: accounts.KindHouse, CurrencyCode: "NGN"},
		{ID: "SUSPENSE-GBP", Kind: accounts.KindSuspense, CurrencyCode: "GBP"},
	}
	for _, w := range want {
		got, err := store.Lookup(w.ID)
		if err != nil {
			t.Fatalf("lookup %s failed: %v", w.ID, err)
		}
		if got != w {
			t.Fatalf("lookup %s: want %+v, got %+v", w.ID, w, got)
		}
	}
}

func TestAccountStoreRejectsDuplicateAndEmptyIDs(t *testing.T) {
	store := NewAccountStore()
	current := &accounts.CurrentAccount{Account: base("CUR-001", "NGN")}

	if err := store.AddCurrent(current); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	err := store.AddCurrent(current)
	if !errors.Is(err, apperrors.ErrAccountExists) {
		t.Fatalf("expected ErrAccountExists, got: %v", err)
	}

	err = store.AddHouse(&accounts.House{})
	if !errors.Is(err, apperrors.ErrAccountIDRequired) {
		t.Fatalf("expected ErrAccountIDRequired, got: %v", err)
	}
}

func TestAccountStoreLookupUnknownAccount(t *testing.T) {
	store := NewAccountStore()

	_, err := store.Lookup("NOPE")
	if !errors.Is(err, apperrors.ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got: %v", err)
	}

	var accountErr *apperrors.AccountError
	if !errors.As(err, &accountErr) || accountErr.AccountID != "NOPE" {
		t.Fatalf("expected an AccountError carrying the id, got: %v", err)
	}
}

func TestAccountStoreConcurrentAccess(t *testing.T) {
	store := NewAccountStore()
	if err := store.AddHouse(&accounts.House{Account: base("HOUSE-NGN", "NGN")}); err != nil {
		t.Fatalf("seeding house: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		id := "CUR-" + strconv.Itoa(i)
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = store.AddCurrent(&accounts.CurrentAccount{Account: base(id, "NGN")})
		}()
		go func() {
			defer wg.Done()
			_, _ = store.Lookup(id)
		}()
		go func() {
			defer wg.Done()
			_, _ = store.FindByKind(accounts.KindHouse, "NGN")
		}()
	}
	wg.Wait()
}

func TestAccountStoreFindByKind(t *testing.T) {
	store := NewAccountStore()
	adds := []error{
		store.AddHouse(&accounts.House{Account: base("HOUSE-NGN", "NGN")}),
		store.AddHouse(&accounts.House{Account: base("HOUSE-USD", "USD")}),
		store.AddSuspense(&accounts.Suspense{Account: base("SUSPENSE-NGN", "NGN")}),
	}
	for i, err := range adds {
		if err != nil {
			t.Fatalf("add %d failed: %v", i, err)
		}
	}

	tests := []struct {
		kind     accounts.Kind
		currency string
		wantID   string
	}{
		{accounts.KindHouse, "NGN", "HOUSE-NGN"},
		{accounts.KindHouse, "USD", "HOUSE-USD"},
		{accounts.KindSuspense, "NGN", "SUSPENSE-NGN"},
	}
	for _, tc := range tests {
		got, err := store.FindByKind(tc.kind, tc.currency)
		if err != nil {
			t.Fatalf("FindByKind(%s, %s): %v", tc.kind, tc.currency, err)
		}
		if got.ID != tc.wantID || got.Kind != tc.kind || got.CurrencyCode != tc.currency {
			t.Fatalf("FindByKind(%s, %s): got %+v, want id %s", tc.kind, tc.currency, got, tc.wantID)
		}
	}
}

func TestAccountStoreFindByKindMisses(t *testing.T) {
	store := NewAccountStore()
	if err := store.AddHouse(&accounts.House{Account: base("HOUSE-NGN", "NGN")}); err != nil {
		t.Fatalf("seeding house: %v", err)
	}
	if err := store.AddCurrent(&accounts.CurrentAccount{Account: base("CUR-001", "NGN")}); err != nil {
		t.Fatalf("seeding current: %v", err)
	}

	misses := []struct {
		name     string
		kind     accounts.Kind
		currency string
	}{
		{"no suspense account at all", accounts.KindSuspense, "NGN"},
		{"no house account in that currency", accounts.KindHouse, "GBP"},
		{"customer kinds are not findable by kind", accounts.KindCustomerCurrent, "NGN"},
	}
	for _, tc := range misses {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.FindByKind(tc.kind, tc.currency)
			if !errors.Is(err, apperrors.ErrNoAccountForKind) {
				t.Fatalf("expected ErrNoAccountForKind, got: %v", err)
			}

			var kindErr *apperrors.KindError
			if !errors.As(err, &kindErr) || kindErr.Kind != string(tc.kind) || kindErr.CurrencyCode != tc.currency {
				t.Fatalf("expected a KindError carrying kind and currency, got: %v", err)
			}
		})
	}
}

func TestAccountStoreCheckCurrency(t *testing.T) {
	store := NewAccountStore()
	if err := store.AddCurrent(&accounts.CurrentAccount{Account: base("CUR-001", "NGN")}); err != nil {
		t.Fatalf("seeding account: %v", err)
	}

	if err := store.CheckCurrency("CUR-001", "NGN"); err != nil {
		t.Fatalf("a matching currency must pass, got: %v", err)
	}

	err := store.CheckCurrency("CUR-001", "USD")
	if !errors.Is(err, apperrors.ErrCurrencyMismatch) {
		t.Fatalf("expected ErrCurrencyMismatch, got: %v", err)
	}

	var mismatch *apperrors.CurrencyMismatchError
	if !errors.As(err, &mismatch) || mismatch.AccountID != "CUR-001" || mismatch.AccountCurrency != "NGN" || mismatch.AmountCurrency != "USD" {
		t.Fatalf("expected the mismatch to carry the account and both currencies, got: %v", err)
	}

	if err := store.CheckCurrency("NOPE", "NGN"); !errors.Is(err, apperrors.ErrAccountNotFound) {
		t.Fatalf("an unknown account must report not found, got: %v", err)
	}
}

func TestAccountStoreRejectsSecondSystemAccountPerCurrency(t *testing.T) {
	store := NewAccountStore()
	if err := store.AddHouse(&accounts.House{Account: base("HOUSE-NGN", "NGN")}); err != nil {
		t.Fatalf("first house: %v", err)
	}

	err := store.AddHouse(&accounts.House{Account: base("HOUSE-NGN-2", "NGN")})
	if !errors.Is(err, apperrors.ErrSystemAccountExists) {
		t.Fatalf("expected ErrSystemAccountExists, got: %v", err)
	}

	if _, err := store.Lookup("HOUSE-NGN-2"); !errors.Is(err, apperrors.ErrAccountNotFound) {
		t.Fatalf("a rejected account must not be stored, got: %v", err)
	}

	found, err := store.FindByKind(accounts.KindHouse, "NGN")
	if err != nil || found.ID != "HOUSE-NGN" {
		t.Fatalf("the original house account must stay findable, got %+v (err: %v)", found, err)
	}

	if err := store.AddSuspense(&accounts.Suspense{Account: base("SUSPENSE-NGN", "NGN")}); err != nil {
		t.Fatalf("a suspense account in the same currency is a different kind and must be allowed: %v", err)
	}
}
