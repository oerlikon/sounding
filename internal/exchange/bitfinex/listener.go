package bitfinex

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"

	"sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

var serverURL = "wss://api-pub.bitfinex.com/ws/2"

type Listener struct {
	symbol string
	opts   Options

	bookCh   atomic.Value
	tradesCh atomic.Value

	ws     *websocket.Conn
	parser fastjson.Parser

	book struct {
		chanID  atomic.Value
		started bool
	}

	trades struct {
		chanID  atomic.Value
		started bool
	}

	subscribed struct {
		mu     sync.Mutex
		book   bool
		trades bool
	}

	nextSeq int64
}

func NewListener(symbol string, options ...exchange.Option) exchange.Listener {
	var opts Options
	for _, o := range options {
		err := o(&opts)
		if err == exchange.ErrCommonOption {
			err = o(&opts.Options)
		}
		if err != nil {
			panic("bitfinex: unknown error setting options")
		}
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
	if l.opts.Logger != nil {
		l.opts.Logger.Printf("Starting listener bitfinex:%s", l.symbol)
	}
	ws, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		return err
	}
	l.ws = ws

	if err := l.sendWsConf(); err != nil {
		return err
	}

	msgs := make(chan []byte, 1)
	go func() {
		errcount := 0
		for {
			if ctx.Err() != nil {
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
			if ctx.Err() != nil {
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
			case <-ctx.Done():
				l.shutdown()
				return
			}
		}
	}()
	return nil
}

func (l *Listener) Book() <-chan *exchange.BookUpdate {
	if bookCh := l.bookCh.Load(); bookCh != nil {
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
	if tradesCh := l.tradesCh.Load(); tradesCh != nil {
		return tradesCh.(chan []*exchange.Trade)
	}
	if !l.subscribed.trades {
		if err := l.subscribeTrades(); err != nil {
			l.err(err)
			return nil
		}
	}
	tradesCh := make(chan []*exchange.Trade, 1)
	l.tradesCh.Store(tradesCh)
	return tradesCh
}

func (l *Listener) err(err error) {
	if l.opts.Logger != nil {
		l.opts.Logger.Println("Error: bitfinex:", err)
	}
}

func (l *Listener) warn(err error) {
	if l.opts.Logger != nil {
		l.opts.Logger.Println("Warning: bitfinex:", err)
	}
}

func (l *Listener) sendWsMessage(msg string) error {
	return l.ws.WriteMessage(websocket.TextMessage, []byte(msg))
}

func (l *Listener) sendWsConf() error {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	TIMESTAMP, SEQ_ALL := 32768, 65536

	msg := fmt.Sprintf(`{"event":"conf","flags":%d}`, TIMESTAMP^SEQ_ALL)
	if err := l.sendWsMessage(msg); err != nil {
		return err
	}
	return nil
}

func (l *Listener) subscribeBook() error {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	if l.subscribed.book {
		return nil
	}
	msg := fmt.Sprintf(`{"event":"subscribe","channel":"book","symbol":"t%s","len":"250"}`,
		strings.ToUpper(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return err
	}
	l.subscribed.book = true
	return nil
}

func (l *Listener) unsubscribeBook() {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	if !l.subscribed.book {
		return
	}
	chanID := l.book.chanID.Load()
	if chanID == nil {
		return
	}
	msg := fmt.Sprintf(`{"event":"unsubscribe","chanId":%d}`, chanID.(int64))
	if err := l.sendWsMessage(msg); err != nil {
		return
	}
	l.book.chanID = atomic.Value{}
	l.subscribed.book = false
	return
}

func (l *Listener) subscribeTrades() error {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	if l.subscribed.trades {
		return nil
	}
	msg := fmt.Sprintf(`{"event":"subscribe","channel":"trades","symbol":"t%s"}`,
		strings.ToUpper(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return err
	}
	l.subscribed.trades = true
	return nil
}

func (l *Listener) unsubscribeTrades() {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	if !l.subscribed.trades {
		return
	}
	chanID := l.trades.chanID.Load()
	if chanID == nil {
		return
	}
	msg := fmt.Sprintf(`{"event":"unsubscribe","chanId":%d}`, chanID.(int64))
	if err := l.sendWsMessage(msg); err != nil {
		return
	}
	l.trades.chanID = atomic.Value{}
	l.subscribed.trades = false
	return
}

func (l *Listener) process(msg []byte) error {
	received := timestamp.Stamp(time.Now())
	v, err := l.parser.ParseBytes(msg)
	if err != nil {
		return err
	}
	if arr, err := v.Array(); err == nil {
		n := len(arr)
		seq, ts := arr[n-2].GetInt64(), timestamp.Milli(arr[n-1].GetInt64())
		if seq != l.nextSeq && l.nextSeq != 0 {
			l.err(fmt.Errorf("missing messages %d..%d", l.nextSeq, seq))
		}
		l.nextSeq = seq + 1

		if bytes.Compare(arr[1].GetStringBytes(), []byte("hb")) == 0 {
			return nil
		}

		chanID := arr[0].GetInt64()
		if id, ok := l.book.chanID.Load().(int64); ok && id == chanID {
			var bu *BookUpdateMessage
			if l.book.started {
				bu = l.parseBookUpdate(arr[1])
			} else {
				bu = l.parseBookSnapshot(arr[1])
				l.book.started = true
			}
			bu.Timestamp, bu.Received = ts, received
			l.sendBookUpdate(bu)
			return nil
		}
		if id, ok := l.trades.chanID.Load().(int64); ok && id == chanID {
			var tu []*TradeMessage
			if l.trades.started {
				teu := arr[1].GetStringBytes()
				if bytes.Compare(teu, []byte("te")) == 0 {
					tu = l.parseTrade(arr[2])
				} else if bytes.Compare(teu, []byte("tu")) != 0 {
					l.err(errors.New(string(msg)))
					return nil
				}
			} else {
				tu = l.parseTradeSnapshot(arr[1])
				l.trades.started = true
			}
			if len(tu) == 1 {
				tu[0].Timestamp, tu[0].Received = ts, received
			} else {
				for i := range tu {
					tu[i].Timestamp, tu[i].Received = ts, received
				}
			}
			l.sendTrades(tu)
			return nil
		}
		return nil
	}
	event := v.GetStringBytes("event")
	if bytes.Compare(event, []byte("subscribed")) == 0 {
		channel := v.GetStringBytes("channel")
		switch {
		case bytes.Compare(channel, []byte("book")) == 0:
			l.book.chanID.Store(v.GetInt64("chanId"))
		case bytes.Compare(channel, []byte("trades")) == 0:
			l.trades.chanID.Store(v.GetInt64("chanId"))
		default:
			return fmt.Errorf("subscribed unexpected channel %s", string(channel))
		}
		return nil
	}
	if bytes.Compare(event, []byte("info")) == 0 {
		return nil
	}
	if bytes.Compare(event, []byte("conf")) == 0 {
		return nil
	}
	if bytes.Index(msg, []byte("error")) >= 0 {
		return errors.New(string(msg))
	}
	l.warn(errors.New(string(msg)))
	return nil
}

func (l *Listener) parseBookSnapshot(v *fastjson.Value) *BookUpdateMessage {
	var bids, asks []exchange.PriceLevelUpdate
	if pqs := v.GetArray(); len(pqs) > 0 {
		bids = make([]exchange.PriceLevelUpdate, 0, len(pqs))
		asks = make([]exchange.PriceLevelUpdate, 0, len(pqs))
		for _, pq := range pqs {
			p := pq.GetArray()[0].S()
			q := pq.GetArray()[2].S()
			if q[0] != '-' { // Bid
				bids = append(bids, exchange.PriceLevelUpdate{
					Price:    p,
					Quantity: q,
				})
			} else {
				asks = append(asks, exchange.PriceLevelUpdate{
					Price:    p,
					Quantity: q[1:],
				})
			}
		}
	}
	return &BookUpdateMessage{
		Bids: bids,
		Asks: asks,
	}
}

func (l *Listener) parseBookUpdate(v *fastjson.Value) *BookUpdateMessage {
	var bids, asks []exchange.PriceLevelUpdate
	if pcq := v.GetArray(); pcq != nil {
		p, c, q := pcq[0].S(), pcq[1].GetInt(), pcq[2].S()
		if c > 0 {
			// Update price level.
			if q[0] != '-' { // Bid
				bids = []exchange.PriceLevelUpdate{
					exchange.PriceLevelUpdate{
						Price:    p,
						Quantity: q,
					},
				}
			} else {
				asks = []exchange.PriceLevelUpdate{
					exchange.PriceLevelUpdate{
						Price:    p,
						Quantity: q[1:],
					},
				}
			}
		} else {
			// Remove price level.
			if q[0] != '-' { // Bid
				bids = []exchange.PriceLevelUpdate{
					exchange.PriceLevelUpdate{
						Price:    p,
						Quantity: "0",
					},
				}
			} else {
				asks = []exchange.PriceLevelUpdate{
					exchange.PriceLevelUpdate{
						Price:    p,
						Quantity: "0",
					},
				}
			}
		}
	}
	return &BookUpdateMessage{
		Bids: bids,
		Asks: asks,
	}
}

func (l *Listener) sendBookUpdate(bu *BookUpdateMessage) {
	bookCh := l.bookCh.Load()
	if bookCh == nil {
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

func (l *Listener) parseTradeSnapshot(v *fastjson.Value) []*TradeMessage {
	tt := v.GetArray()
	if len(tt) == 0 {
		return nil
	}
	trades := make([]*TradeMessage, len(tt))
	for i, t := range tt {
		amount, buy := t.GetArray()[2].S(), true
		if amount[0] == '-' {
			amount, buy = amount[1:], false
		}
		trades[i] = &TradeMessage{
			Occurred: timestamp.Milli(t.GetArray()[1].GetInt64()),
			TradeID:  t.GetArray()[0].GetInt64(),
			Price:    t.GetArray()[3].S(),
			Amount:   amount,
			TakerBuy: buy,
		}
	}
	return trades
}

func (l *Listener) parseTrade(v *fastjson.Value) []*TradeMessage {
	amount, buy := v.GetArray()[2].S(), true
	if amount[0] == '-' {
		amount, buy = amount[1:], false
	}
	return []*TradeMessage{
		&TradeMessage{
			Occurred: timestamp.Milli(v.GetArray()[1].GetInt64()),
			TradeID:  v.GetArray()[0].GetInt64(),
			Price:    v.GetArray()[3].S(),
			Amount:   amount,
			TakerBuy: buy,
		},
	}
}

func (l *Listener) sendTrades(trades []*TradeMessage) {
	tradesCh := l.tradesCh.Load()
	if tradesCh == nil {
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
			TradeID:   trade.TradeID,
			Price:     trade.Price,
			Quantity:  trade.Amount,
			Taker: func() exchange.Side {
				if trade.TakerBuy {
					return exchange.Buy
				}
				return exchange.Sell
			}(),
		}
	}
	tradesCh.(chan []*exchange.Trade) <- tt
}

func (l *Listener) shutdown() {
	if l.opts.Logger != nil {
		l.opts.Logger.Printf("Stopping listener bitfinex:%s", l.symbol)
	}
	if bookCh := l.bookCh.Load(); bookCh != nil {
		l.unsubscribeBook()
		close(bookCh.(chan *exchange.BookUpdate))
		l.bookCh = atomic.Value{}
	}
	if tradesCh := l.tradesCh.Load(); tradesCh != nil {
		l.unsubscribeTrades()
		close(tradesCh.(chan []*exchange.Trade))
		l.tradesCh = atomic.Value{}
	}
	l.ws.Close()
}
