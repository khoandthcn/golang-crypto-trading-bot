package environment

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

//CandleStick represents a single candlestick in a chart.
type CandleStick struct {
	High   decimal.Decimal //Represents the highest value obtained during candle period.
	Open   decimal.Decimal //Represents the first value of the candle period.
	Close  decimal.Decimal //Represents the last value of the candle period.
	Low    decimal.Decimal //Represents the lowest value obtained during candle period.
	Volume decimal.Decimal //Represents the volume of trades during the candle period.
}

// String returns the string representation of the object.
func (cs CandleStick) String() string {
	var color string
	if cs.Open.GreaterThan(cs.Close) {
		color = "Green/Bullish"
	} else if cs.Open.LessThan(cs.Close) {
		color = "Red/Bearish"
	} else {
		color = "Neutral"
	}
	ret := fmt.Sprintln(color, "Candle")
	ret += fmt.Sprintln("High:", cs.High)
	ret += fmt.Sprintln("Open:", cs.Open)
	ret += fmt.Sprintln("Close:", cs.Close)
	ret += fmt.Sprintln("Low:", cs.Low)
	ret += fmt.Sprintln("Volume:", cs.Volume)
	return strings.TrimSpace(ret)
}
