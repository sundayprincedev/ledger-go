package ledger

import (
	"sync"

	"github.com/sundayprincedev/ledger/internal/accounts"
	"github.com/sundayprincedev/ledger/internal/constants"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

type AccountProfile struct {
	ID           string
	Kind         accounts.Kind
	CurrencyCode string
	Tier         constants.KYCTier
}

type AccountStore struct {
	mu       sync.RWMutex
	profiles map[string]AccountProfile
}

func NewAccountStore() *AccountStore {
	return &AccountStore{profiles: make(map[string]AccountProfile)}
}

func (s *AccountStore) AddCurrent(a *accounts.CurrentAccount) error {
	return s.add(AccountProfile{
		ID:           a.AccountID,
		Kind:         accounts.KindCustomerCurrent,
		CurrencyCode: a.CurrencyCode,
		Tier:         a.Tier,
	})
}

func (s *AccountStore) AddSavings(a *accounts.SavingsAccount) error {
	return s.add(AccountProfile{
		ID:           a.AccountID,
		Kind:         accounts.KindCustomerSavings,
		CurrencyCode: a.CurrencyCode,
		Tier:         a.Tier,
	})
}

func (s *AccountStore) AddMerchant(a *accounts.Merchant) error {
	return s.add(AccountProfile{
		ID:           a.AccountID,
		Kind:         accounts.KindMerchant,
		CurrencyCode: a.CurrencyCode,
		Tier:         a.Tier,
	})
}

func (s *AccountStore) AddHouse(a *accounts.House) error {
	return s.add(AccountProfile{
		ID:           a.AccountID,
		Kind:         accounts.KindHouse,
		CurrencyCode: a.CurrencyCode,
	})
}

func (s *AccountStore) AddSuspense(a *accounts.Suspense) error {
	return s.add(AccountProfile{
		ID:           a.AccountID,
		Kind:         accounts.KindSuspense,
		CurrencyCode: a.CurrencyCode,
	})
}

func (s *AccountStore) add(p AccountProfile) error {
	if p.ID == "" {
		return &apperrors.AccountError{AccountID: p.ID, Err: apperrors.ErrAccountIDRequired}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.profiles[p.ID]; exists {
		return &apperrors.AccountError{AccountID: p.ID, Err: apperrors.ErrAccountExists}
	}
	s.profiles[p.ID] = p
	return nil
}

func (s *AccountStore) Lookup(id string) (AccountProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.profiles[id]
	if !ok {
		return AccountProfile{}, &apperrors.AccountError{AccountID: id, Err: apperrors.ErrAccountNotFound}
	}
	return p, nil
}
