package main

import (
	"github.com/sundayprincedev/ledger/internal/accounts"
	"github.com/sundayprincedev/ledger/internal/constants"
	"github.com/sundayprincedev/ledger/internal/data"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

func buildStore(seed []data.SeedAccount) (*ledger.AccountStore, error) {
	store := ledger.NewAccountStore()
	for _, seedAccount := range seed {
		if err := addSeedAccount(store, seedAccount); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func addSeedAccount(store *ledger.AccountStore, seed data.SeedAccount) error {
	base := accounts.Account{
		BaseAccount: accounts.BaseAccount{
			AccountID:    seed.AccountID,
			Kind:         accounts.Kind(seed.Kind),
			CurrencyCode: seed.CurrencyCode,
		},
		Balance: seed.Balance,
	}
	customer := accounts.Customer{
		Name:       seed.CustomerName,
		CustomerID: seed.CustomerID,
		Tier:       constants.KYCTier(seed.Tier),
	}

	switch accounts.Kind(seed.Kind) {
	case accounts.KindCustomerCurrent:
		return store.AddCurrent(&accounts.CurrentAccount{Account: base, Customer: customer, OverdraftLimit: seed.OverdraftLimit})
	case accounts.KindCustomerSavings:
		return store.AddSavings(&accounts.SavingsAccount{Account: base, Customer: customer, InterestRate: seed.InterestRate})
	case accounts.KindMerchant:
		return store.AddMerchant(&accounts.Merchant{
			Account:      base,
			MerchantID:   seed.MerchantID,
			MerchantName: seed.MerchantName,
			Tier:         constants.KYCTier(seed.Tier),
		})
	case accounts.KindHouse:
		return store.AddHouse(&accounts.House{Account: base})
	case accounts.KindSuspense:
		return store.AddSuspense(&accounts.Suspense{Account: base})
	default:
		return &apperrors.KindError{Kind: seed.Kind, CurrencyCode: seed.CurrencyCode, Err: apperrors.ErrUnknownAccountKind}
	}
}

func buildBalances(seed []data.SeedAccount) *ledger.Balances {
	balances := ledger.NewBalances()
	for _, seedAccount := range seed {
		balances.Set(seedAccount.AccountID, seedAccount.CurrencyCode, seedAccount.Balance)
	}
	return balances
}
