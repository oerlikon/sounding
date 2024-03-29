package kraken

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/oerlikon/fastjson"
	"github.com/oerlikon/fastjson/fastfloat"

	. "github.com/oerlikon/sounding/internal/common"
	"github.com/oerlikon/sounding/internal/common/timestamp"
	"github.com/oerlikon/sounding/internal/exchange"
)

const serverURL = "wss://ws.kraken.com"

type Listener struct {
	symbol string
	opts   Options

	ctx    context.Context
	cancel context.CancelFunc

	bookCh   atomic.Value
	tradesCh atomic.Value

	ws     *websocket.Conn
	parser fastjson.Parser

	book struct {
		channelName atomic.Value
		started     bool
	}

	trade struct {
		channelName atomic.Value
	}

	subscribed struct {
		sync.Mutex
		book  bool
		trade bool
	}
}

func NewListener(symbol string, options ...Option) exchange.Listener {
	var opts Options
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			panic("kraken: error setting options: " + err.Error())
		}
	}
	if opts.Stderr == nil {
		opts.Stderr = Silent
	}
	return &Listener{
		symbol: symbol,
		opts:   opts,
	}
}

func (l *Listener) Exchange() string {
	return exchName
}

func (l *Listener) Symbol() string {
	return l.symbol
}

func (l *Listener) Start(ctx context.Context) error {
	l.opts.Stderr.Printf("Starting listener kraken:%s", l.symbol)
	ws, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		return err
	}
	l.ws = ws

	l.ctx, l.cancel = context.WithCancel(ctx)

	msgs := make(chan []byte, 1)
	go func() {
		errcount := 0
		for {
			if l.ctx.Err() != nil {
				return
			}
			_, msg, err := l.ws.ReadMessage()
			if err == nil {
				if len(msg) > 0 {
					msgs <- msg
				}
				errcount = 0
				continue
			}
			if errors.Is(err, net.ErrClosed) && l.ctx.Err() != nil {
				return
			}
			l.err(err)
			if errcount++; errcount == 5 {
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case msg := <-msgs:
				if err := l.process(msg); err != nil {
					l.err(err)
				}
			case <-l.ctx.Done():
				l.shutdown()
				return
			}
		}
	}()
	return nil
}

func (l *Listener) Book() <-chan *exchange.BookUpdate {
	if l.ctx == nil {
		return nil
	}
	if bookCh := l.bookCh.Load(); bookCh != nil && bookCh.(chan *exchange.BookUpdate) != nil {
		return bookCh.(chan *exchange.BookUpdate)
	}
	if !l.subscribed.book {
		if err := l.subscribeBook(); err != nil {
			l.err(err)
			return nil
		}
	}
	bookCh := make(chan *exchange.BookUpdate, 1)
	l.bookCh.Store(bookCh)
	return bookCh
}

func (l *Listener) Trades() <-chan []*exchange.Trade {
	if l.ctx == nil {
		return nil
	}
	if tradesCh := l.tradesCh.Load(); tradesCh != nil && tradesCh.(chan []*exchange.Trade) != nil {
		return tradesCh.(chan []*exchange.Trade)
	}
	if !l.subscribed.trade {
		if err := l.subscribeTrade(); err != nil {
			l.err(err)
			return nil
		}
	}
	tradesCh := make(chan []*exchange.Trade, 1)
	l.tradesCh.Store(tradesCh)
	return tradesCh
}

func (l *Listener) err(err error) {
	l.opts.Stderr.Println("Error: kraken:", err)
}

func (l *Listener) warn(err error) {
	l.opts.Stderr.Println("Warning: kraken:", err)
}

func (l *Listener) sendWsMessage(msg string) error {
	return l.ws.WriteMessage(websocket.TextMessage, []byte(msg))
}

func (l *Listener) subscribeBook() error {
	l.subscribed.Lock()
	defer l.subscribed.Unlock()

	if l.subscribed.book {
		return nil
	}
	msg := fmt.Sprintf(`{"event":"subscribe","pair":["%s"],"subscription":{"name":"book","depth":100}}`,
		strings.ToUpper(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return err
	}
	l.subscribed.book = true
	return nil
}

func (l *Listener) unsubscribeBook() {
	l.subscribed.Lock()
	defer l.subscribed.Unlock()

	if !l.subscribed.book {
		return
	}
	msg := fmt.Sprintf(`{"event":"unsubscribe","pair":["%s"],"subscription":{"name":"book","depth":100}}`,
		strings.ToUpper(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return
	}
	l.book.channelName.Store("")
	l.subscribed.book = false
}

func (l *Listener) subscribeTrade() error {
	l.subscribed.Lock()
	defer l.subscribed.Unlock()

	if l.subscribed.trade {
		return nil
	}
	msg := fmt.Sprintf(`{"event":"subscribe","pair":["%s"],"subscription":{"name":"trade"}}`,
		strings.ToUpper(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return err
	}
	l.subscribed.trade = true
	return nil
}

func (l *Listener) unsubscribeTrade() {
	l.subscribed.Lock()
	defer l.subscribed.Unlock()

	if !l.subscribed.trade {
		return
	}
	msg := fmt.Sprintf(`{"event":"unsubscribe","pair":["%s"],"subscription":{"name":"trade"}}`,
		strings.ToUpper(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return
	}
	l.trade.channelName.Store("")
	l.subscribed.trade = false
}

func (l *Listener) process(msg []byte) error {
	received := timestamp.Stamp(time.Now())
	v, err := l.parser.ParseBytes(msg)
	if err != nil {
		return err
	}
	if arr, err := v.Array(); err == nil {
		channelName := arr[2].S()
		if name, ok := l.book.channelName.Load().(string); ok && name == channelName {
			var bu *BookUpdateMessage
			if l.book.started {
				bu = l.parseBookUpdate(arr[1])
			} else {
				bu = l.parseBookSnapshot(arr[1])
				l.book.started = true
			}
			bu.Timestamp, bu.Received = received, received
			l.sendBookUpdate(bu)
			return nil
		}
		if name, ok := l.trade.channelName.Load().(string); ok && name == channelName {
			tu := l.parseTrade(arr[1])
			for i := range tu {
				tu[i].Timestamp, tu[i].Received = received, received
			}
			l.sendTrades(tu)
			return nil
		}
		return nil
	}
	event := v.GetStringBytes("event")
	if bytes.Equal(event, []byte("subscriptionStatus")) {
		status := v.GetStringBytes("status")
		if bytes.Equal(status, []byte("subscribed")) {
			channel := v.GetStringBytes("subscription", "name")
			channelName := v.Get("channelName").S()
			switch {
			case bytes.Equal(channel, []byte("book")):
				l.book.channelName.Store(channelName)
			case bytes.Equal(channel, []byte("trade")):
				l.trade.channelName.Store(channelName)
			default:
				return fmt.Errorf("subscribed unexpected channel %s", channelName)
			}
			return nil
		}
		// fallthrough
	}
	if bytes.Equal(event, []byte("heartbeat")) {
		return nil
	}
	if bytes.Equal(event, []byte("systemStatus")) {
		return nil
	}
	if bytes.Contains(msg, []byte("error")) {
		return errors.New(string(msg))
	}
	l.warn(errors.New(string(msg)))
	return nil
}

func (l *Listener) parseBookSnapshot(v *fastjson.Value) *BookUpdateMessage {
	var bids, asks []exchange.PriceLevelUpdate

	if bs := v.GetArray("bs"); bs != nil {
		bids = make([]exchange.PriceLevelUpdate, len(bs))
		for i, pq := range bs {
			bids[i].Price = pq.GetArray()[0].S()
			bids[i].Quantity = pq.GetArray()[1].S()
		}
	}
	if as := v.GetArray("as"); as != nil {
		asks = make([]exchange.PriceLevelUpdate, len(as))
		for i, pq := range as {
			asks[i].Price = pq.GetArray()[0].S()
			asks[i].Quantity = pq.GetArray()[1].S()
		}
	}

	return &BookUpdateMessage{
		Bids: bids,
		Asks: asks,
	}
}

func (l *Listener) parseBookUpdate(v *fastjson.Value) *BookUpdateMessage {
	var bids, asks []exchange.PriceLevelUpdate

	if b := v.GetArray("b"); b != nil {
		bids = make([]exchange.PriceLevelUpdate, len(b))
		for i, pq := range b {
			bids[i].Price = pq.GetArray()[0].S()
			bids[i].Quantity = pq.GetArray()[1].S()
		}
	}
	if a := v.GetArray("a"); a != nil {
		asks = make([]exchange.PriceLevelUpdate, len(a))
		for i, pq := range a {
			asks[i].Price = pq.GetArray()[0].S()
			asks[i].Quantity = pq.GetArray()[1].S()
		}
	}

	return &BookUpdateMessage{
		Bids: bids,
		Asks: asks,
	}
}

func (l *Listener) sendBookUpdate(bu *BookUpdateMessage) {
	bookCh := l.bookCh.Load()
	if bookCh == nil || bookCh.(chan *exchange.BookUpdate) == nil {
		return
	}
	bookCh.(chan *exchange.BookUpdate) <- &exchange.BookUpdate{
		Exchange:  exchName,
		Symbol:    l.symbol,
		Timestamp: bu.Timestamp,
		Received:  bu.Received,
		Bids:      bu.Bids,
		Asks:      bu.Asks,
	}
}

func (l *Listener) parseTrade(v *fastjson.Value) []*TradeMessage {
	tt := v.GetArray()
	if len(tt) == 0 {
		return nil
	}
	trades := make([]*TradeMessage, len(tt))
	for i, t := range tt {
		var taker exchange.Side
		switch s := t.GetArray()[3].S(); s {
		case "b":
			taker = exchange.Buy
		case "s":
			taker = exchange.Sell
		default:
			panic(fmt.Sprintf("kraken: unexpected taker '%s'", s))
		}
		trades[i] = &TradeMessage{
			Occurred: timestamp.Float(fastfloat.ParseBestEffort(t.GetArray()[2].S())),
			Price:    t.GetArray()[0].S(),
			Volume:   t.GetArray()[1].S(),
			Taker:    taker,
		}
	}
	return trades
}

func (l *Listener) sendTrades(trades []*TradeMessage) {
	tradesCh := l.tradesCh.Load()
	if tradesCh == nil || tradesCh.(chan []*exchange.Trade) == nil {
		return
	}
	if len(trades) == 0 {
		return
	}
	tt := make([]*exchange.Trade, len(trades))
	for i, trade := range trades {
		tt[i] = &exchange.Trade{
			Exchange:  exchName,
			Symbol:    l.symbol,
			Timestamp: trade.Timestamp,
			Received:  trade.Received,
			Occurred:  trade.Occurred,
			Price:     trade.Price,
			Quantity:  trade.Volume,
			Taker:     trade.Taker,
		}
	}
	tradesCh.(chan []*exchange.Trade) <- tt
}

func (l *Listener) shutdown() {
	l.opts.Stderr.Printf("Stopping listener kraken:%s", l.symbol)
	if bookCh := l.bookCh.Load(); bookCh != nil && bookCh.(chan *exchange.BookUpdate) != nil {
		l.unsubscribeBook()
		close(bookCh.(chan *exchange.BookUpdate))
		l.bookCh.Store((chan *exchange.BookUpdate)(nil))
	}
	if tradesCh := l.tradesCh.Load(); tradesCh != nil && tradesCh.(chan []*exchange.Trade) != nil {
		l.unsubscribeTrade()
		close(tradesCh.(chan []*exchange.Trade))
		l.tradesCh.Store((chan []*exchange.Trade)(nil))
	}
	l.ws.Close()
	l.cancel()
}
