package main

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"sounding/internal/exchange"
)

func Trades(listeners []exchange.Listener) []<-chan []*exchange.Trade {
	trades := make([]<-chan []*exchange.Trade, 0, len(listeners))
	for _, listener := range listeners {
		if listener == nil {
			continue
		}
		if tc := listener.Trades(); tc != nil {
			trades = append(trades, tc)
		}
	}
	if len(trades) == 0 {
		return nil
	}
	return trades
}

func TradesLoop(trades []<-chan []*exchange.Trade, w io.StringWriter, wg *sync.WaitGroup) {
	cases := make([]reflect.SelectCase, len(trades))
	for i, tc := range trades {
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(tc),
		}
	}
	var b strings.Builder
	for len(cases) > 0 {
		n, value, ok := reflect.Select(cases)
		if !ok {
			cases = append(cases[:n], cases[n+1:]...)
			continue
		}
		b.Reset()
		for _, trade := range value.Interface().([]*exchange.Trade) {
			fmt.Fprintf(&b, "T %d,%s,%s,%s,%d,%d,%d,%s,%s,%s\n",
				trade.Occurred.UnixMilli(),
				trade.Occurred.Format("2006-01-02 15:04:05.000"),
				trade.Exchange,
				strings.ToUpper(trade.Symbol),
				trade.TradeID,
				trade.BuyOrderID,
				trade.SellOrderID,
				func() string {
					if trade.Taker == exchange.Buy {
						return "BUY"
					}
					return "SELL"
				}(),
				trade.Price,
				trade.Quantity)
		}
		w.WriteString(b.String())
	}
	wg.Done()
}
