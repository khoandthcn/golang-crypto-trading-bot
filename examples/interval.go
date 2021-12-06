// Copyright © 2017 Alessandro Sanino <saninoale@gmail.com>
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

package examples

import (
	"fmt"
	"time"

	bot "github.com/saniales/golang-crypto-trading-bot/cmd"
	"github.com/saniales/golang-crypto-trading-bot/environment"
	"github.com/saniales/golang-crypto-trading-bot/exchanges"
	"github.com/saniales/golang-crypto-trading-bot/plot"
	"github.com/saniales/golang-crypto-trading-bot/strategies"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
)

// var telegramBot *tb.Bot

var chatGroup *tb.Chat
var telegramBot *tb.Bot
var sendOption = &tb.SendOptions{
	ParseMode: tb.ModeMarkdown,
}

var telegram_enabled = true

// Watch5Sec prints out the info of the market every 5 seconds.
var Watch5Sec = strategies.IntervalStrategy{
	Model: strategies.StrategyModel{
		Name: "Watch5Sec",
		Setup: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
			chatGroup = &tb.Chat{
				ID: bot.BotConfig.TelegramConfig.GroupID,
			}
			telegramBot, _ = tb.NewBot(tb.Settings{
				Token:  bot.BotConfig.TelegramConfig.BotToken,
				Poller: &tb.LongPoller{Timeout: 10 * time.Second},
			})
			telegram_enabled = bot.BotConfig.TelegramConfig.Enabled
			if telegram_enabled {
				msg, err := telegramBot.Send(chatGroup, "Hello every body, I'm ready to go!", sendOption)
				if err != nil {
					logrus.Warning("Failed to send message: " + msg.Text)
				}
			}
			return nil
		},
		OnUpdate: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
			for i, mk := range markets {
				wr := wrappers[0]
				mkSummary, err := wr.GetMarketSummary(markets[i])
				if err != nil {
					return err
				}
				baseBalance, err := wr.GetBalance(mk.BaseCurrency)
				if err != nil {
					return err
				}
				marketBalance, err := wr.GetBalance(mk.MarketCurrency)
				if err != nil {
					return err
				}

				candle, err := wr.GetCandles(mk, "4h")
				if err != nil {
					return err
				}
				candleChart := plot.CandleStickChart{
					CurrentPrice: mkSummary.Last,
					CandlePeriod: time.Hour * 4,
					CandleSticks: candle,
					OrderBook:    nil,
				}
				support := candleChart.GetSupportPrices()
				var action string
				supRange := support[0].Value.Sub(support[len(support)-1].Value)
				if supRange.Equal(decimal.Zero) {
					// find one support only
					if mkSummary.Last.GreaterThan(support[0].Value) &&
						marketBalance.GreaterThanOrEqual(decimal.NewFromInt(5)) {
						action = fmt.Sprintf("BUY at %s", support[0])
					} else if mkSummary.Last.LessThan(support[0].Value) &&
						baseBalance.Mul(mkSummary.Last).GreaterThanOrEqual(decimal.NewFromInt(5)) {
						action = fmt.Sprintf("SELL at %s", support[0])
					} else {
						action = "NOTHING"
					}
				} else {
					position := mkSummary.Last.Sub(support[len(support)-1].Value).
						Div(supRange)
					if position.LessThanOrEqual(decimal.NewFromFloat(0.1)) &&
						marketBalance.GreaterThanOrEqual(decimal.NewFromInt(5)) {
						action = fmt.Sprintf("BUY at %s", support[len(support)-1])
					} else if position.GreaterThanOrEqual(decimal.NewFromFloat(0.9)) &&
						baseBalance.Mul(mkSummary.Last).GreaterThanOrEqual(decimal.NewFromInt(5)) {
						action = fmt.Sprintf("SELL at %s", support[0])
					} else {
						action = "NOTHING"
					}
				}
				logrus.Infof("Market %s-%s: last=%s, Supp=%s\n\tRecommended: %s",
					mk.BaseCurrency, mk.MarketCurrency, mkSummary.Last, support, action)
				if action != "NOTHING" {
					if telegram_enabled {
						msg, err := telegramBot.Send(chatGroup,
							fmt.Sprintf("Market %s-%s: last=%s, Supp=%s\n\tRecommended: %s",
								mk.BaseCurrency, mk.MarketCurrency, mkSummary.Last, support, action),
							sendOption)
						if err != nil {
							logrus.Warning("Failed to send message: " + msg.Text)
						}

						candleChart.ExportPng(fmt.Sprintf("%s%s_candlesticks.png", mk.BaseCurrency, mk.MarketCurrency))

						p := &tb.Photo{File: tb.FromDisk(fmt.Sprintf("%s%s_candlesticks.png", mk.BaseCurrency, mk.MarketCurrency))}
						_, err = telegramBot.Send(chatGroup, p)
						if err != nil {
							logrus.Warning("Failed to upload photo.")
						}

					}
				} else {
					candleChart.ExportPng(fmt.Sprintf("%s%s_candlesticks.png", mk.BaseCurrency, mk.MarketCurrency))
				}

				// elliottModel := ElliottWaveModel(candle[0:100])
				// logrus.Infof("Elliott Wave params: %s", elliottModel)
				// lastCandle := candle[len(candle)-1]
				// logrus.Infof("Last stick: Open: %s, High: %s, Low: %s, Close: %s, Vol: %s",
				// lastCandle.Open, lastCandle.High, lastCandle.Low, lastCandle.Close, lastCandle.Volume)
			}
			return nil
		},
		OnError: func(err error) {
			fmt.Println(err)
		},
		TearDown: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
			fmt.Println("Watch5Sec exited")
			return nil
		},
	},
	Interval: time.Minute * 5,
}

// type SupportPoint struct {
// 	Value  decimal.Decimal
// 	Weight decimal.Decimal
// }

// func (s SupportPoint) String() string {
// 	return fmt.Sprintf("%s(%s)", s.Value, s.Weight)
// }

// func findSupportPoint(candle []environment.CandleStick, last decimal.Decimal) []SupportPoint {
// 	dh := make([]decimal.Decimal, len(candle))
// 	dl := make([]decimal.Decimal, len(candle))
// 	threshold := 0.02
// 	for i := 1; i < len(dh); i++ {
// 		dh[i] = candle[i].High.Sub(candle[i-1].High)
// 		dl[i] = candle[i].Low.Sub(candle[i-1].Low)
// 	}
// 	var criticalPoint []decimal.Decimal
// 	for i := 1; i < len(dh)-1; i++ {
// 		if dh[i].IsNegative() && dh[i+1].IsPositive() {
// 			// local maximal
// 			criticalPoint = append(criticalPoint, candle[i].High)
// 			// decimal.Max(candle[i].Open, candle[i].Close))
// 		} else if dh[i].IsPositive() && dh[i+1].IsNegative() {
// 			// local minimal
// 			criticalPoint = append(criticalPoint, candle[i].High)
// 			// decimal.Max(candle[i].Open, candle[i].Close))
// 		}
// 		if dl[i].IsNegative() && dl[i+1].IsPositive() {
// 			// local maximal
// 			criticalPoint = append(criticalPoint, candle[i].Low)
// 			// decimal.Max(candle[i].Open, candle[i].Close))
// 		} else if dl[i].IsPositive() && dl[i+1].IsNegative() {
// 			// local minimal
// 			criticalPoint = append(criticalPoint, candle[i].Low)
// 			// decimal.Max(candle[i].Open, candle[i].Close))
// 		}
// 	}
// 	sort.Slice(criticalPoint, func(i, j int) bool {
// 		return criticalPoint[i].GreaterThan(criticalPoint[j])
// 	})
// 	supportPoint := []SupportPoint{}
// 	sum := criticalPoint[0]
// 	count := decimal.NewFromInt(1)
// 	for i := 1; i < len(criticalPoint); i++ {
// 		if sum.Div(count).Sub(criticalPoint[i]).Div(sum.Div(count)).LessThanOrEqual(decimal.NewFromFloat(threshold)) {
// 			sum = sum.Add(criticalPoint[i])
// 			count = count.Add(decimal.NewFromInt(1))
// 		} else {
// 			supportPoint = append(supportPoint, SupportPoint{Value: sum.Div(count), Weight: count})
// 			sum = criticalPoint[i]
// 			count = decimal.NewFromInt(1)
// 		}
// 	}
// 	logrus.Infof("Support: %s", supportPoint)
// 	startIdx := 0
// 	endIdx := 0
// 	for i := 0; i < len(supportPoint); i++ {
// 		if last.LessThan(supportPoint[i].Value.Mul(decimal.NewFromFloat(1))) &&
// 			supportPoint[i].Weight.GreaterThanOrEqual(decimal.NewFromInt(3)) {
// 			startIdx = i
// 		}
// 		if last.GreaterThan(supportPoint[i].Value.Mul(decimal.NewFromFloat(1))) &&
// 			supportPoint[i].Weight.GreaterThanOrEqual(decimal.NewFromInt(3)) && endIdx == 0 {
// 			endIdx = i
// 			break
// 		}
// 	}
// 	return supportPoint[startIdx : endIdx+1]
// }

// func exportPng(candle []environment.CandleStick, support []SupportPoint, fileName string) error {
// 	candleSticks, err := plot.NewCandlesticks(candle)
// 	if err != nil {
// 		return err
// 	}
// 	p := pl.New()
// 	p.Title.Text = "Candlesticks"
// 	p.X.Label.Text = "Time"
// 	p.Y.Label.Text = "Price"
// 	// p.X.Tick.Marker = pl.TimeTicks{Format: "2006-01-02\n15:04:05"}

// 	p.Add(candleSticks)

// 	for i := 0; i < len(support); i++ {
// 		if support[i].Weight.GreaterThanOrEqual(decimal.NewFromInt(3)) {
// 			value, _ := support[i].Value.Float64()
// 			err = plotutil.AddLines(p, fmt.Sprintf("S(%s)", support[i].Weight), HorizontalLine(len(candle), value))
// 			if err != nil {
// 				panic(err)
// 			}
// 		}
// 	}

// 	logrus.Info("Find trendline")
// 	plotutil.AddLines(p, "Trend Line", TrendLine(candle))

// 	logrus.Info("done")
// 	err = p.Save(1024, 768, fileName)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func TrendLine(candle []environment.CandleStick) plotter.XYs {
// 	xTrain := make([]float64, len(candle))
// 	yTrain := make([]float64, len(candle))
// 	for i := 0; i < len(candle); i++ {
// 		xTrain[i] = float64(i)
// 		yTrain[i], _ = candle[i].High.Float64()
// 	}
// 	// Fit
// 	lr := optimize.LinearRegression{NIter: 100, Method: "gd"}
// 	lr.Fit(xTrain, yTrain)
// 	logrus.Printf("Trendline %s", lr.Weights)
// 	yPredict := lr.Predict(xTrain)
// 	pts := make(plotter.XYs, len(candle))
// 	for i := 0; i < len(candle); i++ {
// 		pts[i].X = xTrain[i]
// 		pts[i].Y = yPredict[i]
// 	}

// 	return pts
// }

// func HorizontalLine(n int, h float64) plotter.XYs {
// 	pts := make(plotter.XYs, n)
// 	for i := range pts {
// 		pts[i].X = float64(i)
// 		pts[i].Y = h
// 	}
// 	return pts
// }
