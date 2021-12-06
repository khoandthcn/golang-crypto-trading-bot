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
				wr.GetMarkets()
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
	Interval: time.Minute * 1,
}
