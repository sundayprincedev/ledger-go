package money

import "math"

type Currency struct {
	Code          string
	Name          string
	DecimalPlaces int
}

var Currencies = map[string]Currency{
	"USD": {Code: "USD", Name: "US Dollar", DecimalPlaces: 2},
	"NGN": {Code: "NGN", Name: "Nigerian Naira", DecimalPlaces: 2},
	"EUR": {Code: "EUR", Name: "Euro", DecimalPlaces: 2},
	"GBP": {Code: "GBP", Name: "British Pound", DecimalPlaces: 2},
	"JPY": {Code: "JPY", Name: "Japanese Yen", DecimalPlaces: 0},
	"CAD": {Code: "CAD", Name: "Canadian Dollar", DecimalPlaces: 2},
	"AUD": {Code: "AUD", Name: "Australian Dollar", DecimalPlaces: 2},
}

func CurrencyExists(code string) (Currency, bool) {
	c, ok := Currencies[code]
	return c, ok
}

func scale(decimalPlaces int) float64 {
	return math.Pow(10, float64(decimalPlaces))
}

func NormalizeAmountByRound(amount float64, curr Currency) int64 {
	return int64(math.Round(amount * scale(curr.DecimalPlaces)))
}

func NormalizeAmountByFloor(amount float64, curr Currency) int64 {
	return int64(math.Floor(amount * scale(curr.DecimalPlaces)))
}

func DenormalizeAmount(amount int64, curr Currency) float64 {
	return float64(amount) / scale(curr.DecimalPlaces)
}
