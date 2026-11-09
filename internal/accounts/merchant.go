package accounts

import "github.com/sundayprincedev/ledger/internal/constants"

type Merchant struct {
	Account
	MerchantID   string
	MerchantName string
	Tier         constants.KYCTier
}
