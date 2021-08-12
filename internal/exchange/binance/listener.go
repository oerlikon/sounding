package binance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"

	. "sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

var serverURL = "wss://stream.binance.com:9443/stream"

type Listener struct {
	symbol string
	opts   Options

	bookCh   atomic.Value
	tradesCh atomic.Value

	ws     *websocket.Conn
	parser fastjson.Parser

	depth struct {
		nextID   int64
		updates  []*DepthUpdateMessage
		snapshot atomic.Value
		started  bool
	}

	subscribed struct {
		mu    sync.Mutex
		depth bool
		trade bool
	}
}

func NewListener(symbol string, options ...exchange.Option) exchange.Listener {
	var opts Options
	for _, o := range options {
		err := o(&opts)
		if err == exchange.ErrCommonOption {
			err = o(&opts.Options)
		}
		if err != nil {
			panic("unknown error setting options")
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
		l.opts.Logger.Printf("Starting listener binance:%s", l.symbol)
	}
	ws, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		return err
	}
	l.ws = ws

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
	if !l.subscribed.depth {
		if err := l.subscribeDepth(); err != nil {
			l.err(err)
			return nil
		}
		go l.fetchDepthSnapshot()
	}
	bookCh := make(chan *exchange.BookUpdate, 1)
	l.bookCh.Store(bookCh)
	return bookCh
}

func (l *Listener) Trades() <-chan []*exchange.Trade {
	if tradesCh := l.tradesCh.Load(); tradesCh != nil {
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
	if l.opts.Logger != nil {
		l.opts.Logger.Println("Error:", err)
	}
}

func (l *Listener) warn(err error) {
	if l.opts.Logger != nil {
		l.opts.Logger.Println("Warning:", err)
	}
}

func (l *Listener) sendWsMessage(msg string) error {
	return l.ws.WriteMessage(websocket.TextMessage, []byte(msg))
}

func (l *Listener) subscribeDepth() error {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	if l.subscribed.depth {
		return nil
	}
	msg := fmt.Sprintf(`{"method":"SUBSCRIBE","params":["%s@depth"],"id":1}`,
		strings.ToLower(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return err
	}
	l.subscribed.depth = true
	return nil
}

func (l *Listener) unsubscribeDepth() {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	if !l.subscribed.depth {
		return
	}
	msg := fmt.Sprintf(`{"method":"UNSUBSCRIBE","params":["%s@depth"],"id":1}`,
		strings.ToLower(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return
	}
	l.subscribed.depth = false
	return
}

func (l *Listener) subscribeTrade() error {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	if l.subscribed.trade {
		return nil
	}
	msg := fmt.Sprintf(`{"method":"SUBSCRIBE","params":["%s@trade"],"id":2}`,
		strings.ToLower(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return err
	}
	l.subscribed.trade = true
	return nil
}

func (l *Listener) unsubscribeTrade() {
	l.subscribed.mu.Lock()
	defer l.subscribed.mu.Unlock()

	if !l.subscribed.trade {
		return
	}
	msg := fmt.Sprintf(`{"method":"UNSUBSCRIBE","params":["%s@trade"],"id":2}`,
		strings.ToLower(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return
	}
	l.subscribed.trade = false
	return
}

func (l *Listener) fetchDepthSnapshot() {
	url := fmt.Sprintf("https://api.binance.com/api/v3/depth?symbol=%s&limit=1000",
		strings.ToUpper(l.symbol))
	resp, err := http.Get(url)
	if err != nil {
		l.err(err)
		return
	}
	received := Stamp(time.Now())
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.err(err)
		return
	}

	if bytes.Index(body, []byte("Illegal")) >= 0 {
		l.err(errors.New(string(body)))
		return
	}
	if bytes.Index(body, []byte("Invalid")) >= 0 {
		l.err(errors.New(string(body)))
		return
	}

	var parser fastjson.Parser
	v, err := parser.ParseBytes(body)
	if err != nil {
		l.err(err)
		return
	}
	ds := l.parseDepthSnapshot(v)
	ds.Timestamp, ds.Received = received, received
	l.depth.snapshot.Store(ds)
}

func (l *Listener) process(msg []byte) error {
	received := Stamp(time.Now())
	v, err := l.parser.ParseBytes(msg)
	if err != nil {
		return err
	}
	if stream := v.GetStringBytes("stream"); stream != nil {
		if bytes.HasSuffix(stream, []byte("depth")) {
			du := l.parseDepthUpdate(v.Get("data"))
			du.Received = received

			if l.depth.started {
				l.sendDepthUpdate(du)
				return nil
			}
			l.depth.updates = append(l.depth.updates, du)
			if depthSnapshot := l.depth.snapshot.Load(); depthSnapshot != nil {
				ds := depthSnapshot.(*DepthUpdateMessage)
				l.sendDepthUpdate(ds)
				for _, du := range l.depth.updates {
					if du.FinalID < ds.FinalID+1 {
						continue
					}
					l.sendDepthUpdate(du)
				}
				l.depth.updates, l.depth.snapshot = nil, atomic.Value{}
				l.depth.started = true
			}
			return nil
		}
		if bytes.HasSuffix(stream, []byte("trade")) {
			trade := l.parseTrade(v.Get("data"))
			trade.Received = received
			l.sendTrade(trade)
			return nil
		}
	}
	if bytes.Index(msg, []byte("error")) >= 0 {
		return errors.New(string(msg))
	}
	return nil
}

func (l *Listener) parseDepthSnapshot(v *fastjson.Value) *DepthUpdateMessage {
	var bids, asks []exchange.PriceLevelUpdate

	if b := v.GetArray("bids"); b != nil {
		bids = make([]exchange.PriceLevelUpdate, len(b))
		for i, pq := range b {
			bids[i].Price = pq.GetArray()[0].S()
			bids[i].Quantity = pq.GetArray()[1].S()
		}
	}
	if a := v.GetArray("asks"); a != nil {
		asks = make([]exchange.PriceLevelUpdate, len(a))
		for i, pq := range a {
			asks[i].Price = pq.GetArray()[0].S()
			asks[i].Quantity = pq.GetArray()[1].S()
		}
	}

	return &DepthUpdateMessage{
		FinalID: v.GetInt64("lastUpdateId"),
		Bids:    bids,
		Asks:    asks,
	}
}

func (l *Listener) parseDepthUpdate(v *fastjson.Value) *DepthUpdateMessage {
	firstID, finalID := v.GetInt64("U"), v.GetInt64("u")
	if l.depth.nextID != 0 && l.depth.nextID != firstID {
		l.warn(fmt.Errorf("missing depth updates %d:%d",
			l.depth.nextID, firstID))
	}
	l.depth.nextID = finalID + 1

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

	return &DepthUpdateMessage{
		Timestamp: Milli(v.GetInt64("E")),
		FirstID:   firstID,
		FinalID:   finalID,
		Bids:      bids,
		Asks:      asks,
	}
}

func (l *Listener) sendDepthUpdate(du *DepthUpdateMessage) {
	bookCh := l.bookCh.Load()
	if bookCh == nil {
		return
	}
	bookCh.(chan *exchange.BookUpdate) <- &exchange.BookUpdate{
		Exchange:  exchName,
		Symbol:    l.symbol,
		Timestamp: du.Timestamp,
		Received:  du.Received,
		Bids:      du.Bids,
		Asks:      du.Asks,
	}
}

func (l *Listener) parseTrade(v *fastjson.Value) *TradeMessage {
	return &TradeMessage{
		Timestamp:   Milli(v.GetInt64("E")),
		Occurred:    Milli(v.GetInt64("T")),
		TradeID:     v.GetInt64("t"),
		BuyOrderID:  v.GetInt64("b"),
		SellOrderID: v.GetInt64("a"),
		Price:       v.Get("p").S(),
		Quantity:    v.Get("q").S(),
		MakerBuy:    v.GetBool("m"),
	}
}

func (l *Listener) sendTrade(trade *TradeMessage) {
	tradesCh := l.tradesCh.Load()
	if tradesCh == nil {
		return
	}
	tradesCh.(chan []*exchange.Trade) <- []*exchange.Trade{
		&exchange.Trade{
			Exchange:    exchName,
			Symbol:      l.symbol,
			Timestamp:   trade.Timestamp,
			Received:    trade.Received,
			Occurred:    trade.Occurred,
			TradeID:     trade.TradeID,
			BuyOrderID:  trade.BuyOrderID,
			SellOrderID: trade.SellOrderID,
			Price:       trade.Price,
			Quantity:    trade.Quantity,
			Taker: func() exchange.Side {
				if trade.MakerBuy {
					return exchange.Sell
				}
				return exchange.Buy
			}(),
		},
	}
}

func (l *Listener) shutdown() {
	if l.opts.Logger != nil {
		l.opts.Logger.Printf("Stopping listener binance:%s", l.symbol)
	}
	if bookCh := l.bookCh.Load(); bookCh != nil {
		l.unsubscribeDepth()
		close(bookCh.(chan *exchange.BookUpdate))
		l.bookCh = atomic.Value{}
	}
	if tradesCh := l.tradesCh.Load(); tradesCh != nil {
		l.unsubscribeTrade()
		close(tradesCh.(chan []*exchange.Trade))
		l.tradesCh = atomic.Value{}
	}
	l.ws.Close()
}
