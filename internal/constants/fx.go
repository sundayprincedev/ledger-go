package constants

const FXRateScale int64 = 1_000_000

const FXBaseCurrency = "USD"

var FXPerBase = map[string]int64{
	"USD": 1 * FXRateScale,
	"NGN": 1500 * FXRateScale,
	"EUR": 920_000,
	"GBP": 790_000,
	"JPY": 150 * FXRateScale,
	"CAD": 1_360_000,
	"AUD": 1_520_000,
}
