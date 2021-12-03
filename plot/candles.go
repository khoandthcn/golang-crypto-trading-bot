// Copyright © 2021 Nguyen Dang Khoa <khoa.nd.thcn@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package plot

//CandleStick represents a single candle in the graph.
import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/openacid/slim/polyfit"
	"github.com/saniales/golang-crypto-trading-bot/environment"
	"github.com/saniales/golang-crypto-trading-bot/optimize"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	pl "gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
)

//CandleStickChart represents a chart of a market expresed using Candle Sticks.
type CandleStickChart struct {
	CurrentPrice decimal.Decimal
	CandlePeriod time.Duration             //Represents the candle period (expressed in time.Duration).
	CandleSticks []environment.CandleStick //Represents the last Candle Sticks used for evaluation of current state.
	OrderBook    []environment.Order       //Represents the Book of current trades.
}

type SupportPrice struct {
	Value  decimal.Decimal
	Weight decimal.Decimal
}

func (s SupportPrice) String() string {
	return fmt.Sprintf("%s(%s)", s.Value.Round(2), s.Weight)
}

func (csc CandleStickChart) GetSupportPrices() []SupportPrice {
	candle := csc.CandleSticks
	dh := make([]decimal.Decimal, len(candle))
	dl := make([]decimal.Decimal, len(candle))
	threshold := 0.02
	for i := 1; i < len(dh); i++ {
		dh[i] = candle[i].High.Sub(candle[i-1].High)
		dl[i] = candle[i].Low.Sub(candle[i-1].Low)
	}
	var criticalPoint []decimal.Decimal
	for i := 1; i < len(dh)-1; i++ {
		if dh[i].IsNegative() && dh[i+1].IsPositive() {
			// local maximal
			criticalPoint = append(criticalPoint, candle[i].High)
			// decimal.Max(candle[i].Open, candle[i].Close))
		} else if dh[i].IsPositive() && dh[i+1].IsNegative() {
			// local minimal
			criticalPoint = append(criticalPoint, candle[i].High)
			// decimal.Max(candle[i].Open, candle[i].Close))
		}
		if dl[i].IsNegative() && dl[i+1].IsPositive() {
			// local maximal
			criticalPoint = append(criticalPoint, candle[i].Low)
			// decimal.Max(candle[i].Open, candle[i].Close))
		} else if dl[i].IsPositive() && dl[i+1].IsNegative() {
			// local minimal
			criticalPoint = append(criticalPoint, candle[i].Low)
			// decimal.Max(candle[i].Open, candle[i].Close))
		}
	}
	sort.Slice(criticalPoint, func(i, j int) bool {
		return criticalPoint[i].GreaterThan(criticalPoint[j])
	})
	supportPoint := []SupportPrice{}
	sum := criticalPoint[0]
	count := decimal.NewFromInt(1)
	for i := 1; i < len(criticalPoint); i++ {
		if sum.Div(count).Sub(criticalPoint[i]).Div(sum.Div(count)).LessThanOrEqual(decimal.NewFromFloat(threshold)) {
			sum = sum.Add(criticalPoint[i])
			count = count.Add(decimal.NewFromInt(1))
		} else {
			supportPoint = append(supportPoint, SupportPrice{Value: sum.Div(count), Weight: count})
			sum = criticalPoint[i]
			count = decimal.NewFromInt(1)
		}
	}
	logrus.Infof("Support: %s", supportPoint)
	startIdx := 0
	endIdx := 0
	for i := 0; i < len(supportPoint); i++ {
		if csc.CurrentPrice.LessThan(supportPoint[i].Value.Mul(decimal.NewFromFloat(1))) &&
			supportPoint[i].Weight.GreaterThanOrEqual(decimal.NewFromInt(3)) {
			startIdx = i
		}
		if csc.CurrentPrice.GreaterThan(supportPoint[i].Value.Mul(decimal.NewFromFloat(1))) &&
			supportPoint[i].Weight.GreaterThanOrEqual(decimal.NewFromInt(3)) && endIdx == 0 {
			endIdx = i
			break
		}
	}
	return supportPoint[startIdx : endIdx+1]
}

func (csc CandleStickChart) ExportPng(fileName string) error {
	candleSticks, err := NewCandlesticks(csc.CandleSticks)
	if err != nil {
		return err
	}
	p := pl.New()
	p.Title.Text = "Candlesticks"
	p.X.Label.Text = "Time"
	p.Y.Label.Text = "Price"
	// p.X.Tick.Marker = pl.TimeTicks{Format: "2006-01-02\n15:04:05"}

	p.Add(candleSticks)

	support := csc.GetSupportPrices()
	for i := 0; i < len(support); i++ {
		if support[i].Weight.GreaterThanOrEqual(decimal.NewFromInt(3)) {
			value, _ := support[i].Value.Float64()
			err = plotutil.AddLines(p, fmt.Sprintf("S(%s)", support[i].Weight), HorizontalLine(len(csc.CandleSticks), value))
			if err != nil {
				panic(err)
			}
		}
	}

	logrus.Info("Find trendline")
	plotutil.AddLines(p,
		"Trend Line", csc.GetTrendLine(),
		"Elliottt Wave", csc.GetElliottWaveModel())

	logrus.Info("done")
	err = p.Save(1024, 768, fileName)
	if err != nil {
		return err
	}
	return nil
}

func (csc CandleStickChart) GetTrendLine() plotter.XYs {
	candle := csc.CandleSticks
	xTrain := make([]float64, len(candle))
	yTrain := make([]float64, len(candle))
	for i := 0; i < len(candle); i++ {
		xTrain[i] = float64(i)
		yTrain[i], _ = candle[i].High.Float64()
	}
	// Fit
	lr := optimize.LinearRegression{NIter: 100, Method: "gd"}
	lr.Fit(xTrain, yTrain)
	logrus.Printf("Trendline %s", lr.Weights)
	yPredict := lr.Predict(xTrain)
	pts := make(plotter.XYs, len(candle))
	for i := 0; i < len(candle); i++ {
		pts[i].X = xTrain[i]
		pts[i].Y = yPredict[i]
	}

	return pts
}

func (csc CandleStickChart) GetElliottWaveModel() plotter.XYs {
	var xs, ys []float64
	candle := csc.CandleSticks
	xs = make([]float64, len(candle))
	ys = make([]float64, len(candle))
	for i := 0; i < len(candle); i++ {
		xs[i] = float64(i)
		ys[i], _ = candle[i].High.Float64()
	}
	polyfit := polyfit.NewFitting(xs, ys, 10)
	elliottModel := polyfit.Solve(true)

	pts := make(plotter.XYs, len(candle))
	for i := 0; i < len(candle); i++ {
		pts[i].X = float64(i)
		pts[i].Y = elliottModel[0]
		for j := 1; j < len(elliottModel); j++ {
			pts[i].Y += elliottModel[j] * math.Pow(float64(i), float64(j))
		}
	}
	return pts
}

func HorizontalLine(n int, h float64) plotter.XYs {
	pts := make(plotter.XYs, n)
	for i := range pts {
		pts[i].X = float64(i)
		pts[i].Y = h
	}
	return pts
}
