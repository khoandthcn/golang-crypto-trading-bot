// Copyright © 2017 Nguyen Dang Khoa <khoa.nd.thcn@gmail.com>
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

import (
	"image/color"
	"math"

	"github.com/openacid/slim/polyfit"
	"github.com/saniales/golang-crypto-trading-bot/environment"
	"github.com/shopspring/decimal"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

//CandleSticks represents an array of candle in the graph.

// DefaultCandleWidthFactor is the default width of the candle relative to the DefaultLineStyle.Width.
var DefaultCandleWidthFactor = 3

type CandleSticks struct {
	candles []environment.CandleStick

	elliottWave []float64

	// ColorUp is the color of sticks where C >= O
	ColorUp color.Color

	// ColorDown is the color of sticks where C < O
	ColorDown color.Color

	// LineStyle is the style used to draw the sticks.
	draw.LineStyle

	// CandleWidth is the width of a candlestick
	CandleWidth vg.Length

	// FixedLineColor determines if a fixed line color can be used for up and down bars.
	// When set to true then the color of LineStyle is used to draw the sticks and
	// the borders of the candle. If set to false then ColorUp or ColorDown are used to
	// draw the sticks and the borders of the candle. Thus a candle's fill color is also
	// used for the borders and sticks.
	FixedLineColor bool
}

func ElliottWaveModel(candle []environment.CandleStick) []float64 {
	var xs, ys []float64
	xs = make([]float64, len(candle))
	ys = make([]float64, len(candle))
	for i := 0; i < len(candle); i++ {
		xs[i] = float64(i)
		ys[i], _ = candle[i].High.Float64()
	}
	polyfit := polyfit.NewFitting(xs, ys, 10)
	elliottModel := polyfit.Solve(true)
	return elliottModel
}

// NewCandlesticks creates as new candlestick plotter for
// the given data.
func NewCandlesticks(candles []environment.CandleStick) (*CandleSticks, error) {
	elliottModel := ElliottWaveModel(candles)
	ewave := make([]float64, len(candles))
	for i := 0; i < len(candles); i++ {
		ewave[i] = elliottModel[0]
		for j := 1; j < len(elliottModel); j++ {
			ewave[i] += elliottModel[j] * math.Pow(float64(i), float64(j))
		}
	}
	return &CandleSticks{
		candles:        candles,
		elliottWave:    ewave,
		FixedLineColor: true,
		ColorUp:        color.RGBA{R: 128, G: 192, B: 128, A: 255}, // eye is more sensible to green
		ColorDown:      color.RGBA{R: 255, G: 128, B: 128, A: 255},
		LineStyle:      plotter.DefaultLineStyle,
		CandleWidth:    vg.Length(DefaultCandleWidthFactor) * plotter.DefaultLineStyle.Width,
	}, nil
}

// Plot implements the Plot method of the plot.Plotter interface.
func (sticks *CandleSticks) Plot(c draw.Canvas, plt *plot.Plot) {
	trX, trY := plt.Transforms(&c)
	lineStyle := sticks.LineStyle

	for i, candle := range sticks.candles {
		var fillColor color.Color
		if candle.Close.GreaterThanOrEqual(candle.Open) {
			fillColor = sticks.ColorUp
		} else {
			fillColor = sticks.ColorDown
		}

		if !sticks.FixedLineColor {
			lineStyle.Color = fillColor
		}
		// Transform the data
		// to the corresponding drawing coordinate.
		x := trX(float64(i)) //TODO compute stick time
		high, _ := candle.High.Float64()
		yh := trY(high)
		low, _ := candle.Low.Float64()
		yl := trY(low)
		maxoc, _ := decimal.Max(candle.Open, candle.Close).Float64()
		minoc, _ := decimal.Min(candle.Open, candle.Close).Float64()
		ymaxoc := trY(maxoc)
		yminoc := trY(minoc)

		// top stick
		line := c.ClipLinesY([]vg.Point{{x, yh}, {x, ymaxoc}})
		c.StrokeLines(lineStyle, line...)

		// bottom stick
		line = c.ClipLinesY([]vg.Point{{x, yl}, {x, yminoc}})
		c.StrokeLines(lineStyle, line...)

		// body
		poly := c.ClipPolygonY([]vg.Point{{x - sticks.CandleWidth/2, ymaxoc}, {x + sticks.CandleWidth/2, ymaxoc}, {x + sticks.CandleWidth/2, yminoc}, {x - sticks.CandleWidth/2, yminoc}, {x - sticks.CandleWidth/2, ymaxoc}})
		c.FillPolygon(fillColor, poly)
		c.StrokeLines(lineStyle, poly)

		// Elliott wave
		xs := trX(float64(i))
		ys := trY(sticks.elliottWave[i])
		if i > 0 {
			xs = trX(float64(i - 1))
			ys = trY(sticks.elliottWave[i-1])
		}
		xe := trX(float64(i))
		ye := trY(sticks.elliottWave[i])
		line = c.ClipLinesXY([]vg.Point{{xs, ys}, {xe, ye}})
		c.StrokeLines(lineStyle, line...)
	}

	for i := len(sticks.candles); i < len(sticks.candles); i++ {
		xs := trX(float64(i))
		ys := trY(sticks.elliottWave[i])
		if i > 0 {
			xs = trX(float64(i - 1))
			ys = trY(sticks.elliottWave[i-1])
		}
		xe := trX(float64(i))
		ye := trY(sticks.elliottWave[i])
		line := c.ClipLinesXY([]vg.Point{{xs, ys}, {xe, ye}})
		c.StrokeLines(lineStyle, line...)
	}
}

// DataRange implements the DataRange method
// of the plot.DataRanger interface.
func (sticks *CandleSticks) DataRange() (xMin, xMax, yMin, yMax float64) {
	xMin = math.Inf(1)
	xMax = math.Inf(-1)
	yMin = math.Inf(1)
	yMax = math.Inf(-1)
	for i, candle := range sticks.candles {
		xMin = math.Min(xMin, float64(i)) //TODO compute stick time
		xMax = math.Max(xMax, float64(i))
		low, _ := candle.Low.Float64()
		yMin = math.Min(yMin, low)
		high, _ := candle.High.Float64()
		yMax = math.Max(yMax, high)
	}

	return
}

// GlyphBoxes implements the GlyphBoxes method
// of the plot.GlyphBoxer interface.
// We just return 2 glyph boxes at xmin, ymin and xmax, ymax
// Important is that they provide space for the left part of the first candle's body and for the right part of the last candle's body
func (sticks *CandleSticks) GlyphBoxes(plt *plot.Plot) []plot.GlyphBox {
	boxes := make([]plot.GlyphBox, 2)

	xmin, xmax, ymin, ymax := sticks.DataRange()

	boxes[0].X = plt.X.Norm(xmin)
	boxes[0].Y = plt.Y.Norm(ymin)
	boxes[0].Rectangle = vg.Rectangle{
		Min: vg.Point{X: -(sticks.CandleWidth + sticks.LineStyle.Width) / 2, Y: 0},
		Max: vg.Point{X: 0, Y: 0},
	}

	boxes[1].X = plt.X.Norm(xmax)
	boxes[1].Y = plt.Y.Norm(ymax)
	boxes[1].Rectangle = vg.Rectangle{
		Min: vg.Point{X: 0, Y: 0},
		Max: vg.Point{X: +(sticks.CandleWidth + sticks.LineStyle.Width) / 2, Y: 0},
	}

	return boxes
}
