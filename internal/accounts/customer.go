package accounts

import "github.com/sundayprincedev/ledger/internal/constants"

type Customer struct {
	Name       string
	CustomerID string
	Tier       constants.KYCTier
}
