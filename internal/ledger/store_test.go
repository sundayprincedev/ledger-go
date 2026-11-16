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

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		id := "HOUSE-" + strconv.Itoa(i)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = store.AddHouse(&accounts.House{Account: base(id, "NGN")})
		}()
		go func() {
			defer wg.Done()
			_, _ = store.Lookup(id)
		}()
	}
	wg.Wait()
}
