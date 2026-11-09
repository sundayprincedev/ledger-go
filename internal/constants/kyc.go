package constants

type KYCTier int

const (
	TierOne   KYCTier = 1
	TierTwo   KYCTier = 2
	TierThree KYCTier = 3
)

type TierLimits struct {
	SingleTransferLimit int64
	DailyTransferLimit  int64
}

var TierLimitsByTier = map[KYCTier]TierLimits{
	TierOne:   {SingleTransferLimit: 50000, DailyTransferLimit: 150000},
	TierTwo:   {SingleTransferLimit: 500000, DailyTransferLimit: 2000000},
	TierThree: {SingleTransferLimit: 5000000, DailyTransferLimit: 20000000},
}

func LimitsFor(tier KYCTier) (TierLimits, bool) {
	limits, ok := TierLimitsByTier[tier]
	return limits, ok
}
