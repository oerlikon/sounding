package binance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/oerlikon/fastjson"

	. "github.com/oerlikon/sounding/internal/common"
	"github.com/oerlikon/sounding/internal/common/timestamp"
	"github.com/oerlikon/sounding/internal/exchange"
)

const serverURL = "wss://stream.binance.com:9443/stream"

type Listener struct {
	symbol string
	opts   Options

	ctx    context.Context
	cancel context.CancelFunc

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
		sync.Mutex
		depth bool
		trade bool
	}
}

func NewListener(symbol string, options ...Option) exchange.Listener {
	var opts Options
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			panic("binance: error setting options: " + err.Error())
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
	l.opts.Stderr.Printf("Starting listener binance:%s", l.symbol)
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
	if !l.subscribed.depth {
		if err := l.subscribeDepth(); err != nil {
			l.err(err)
			return nil
		}
		go l.fetchDepthSnapshot(l.ctx)
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
	l.opts.Stderr.Println("Error: binance:", err)
}

func (l *Listener) warn(err error) {
	l.opts.Stderr.Println("Warning: binance:", err)
}

func (l *Listener) sendWsMessage(msg string) error {
	return l.ws.WriteMessage(websocket.TextMessage, []byte(msg))
}

func (l *Listener) subscribeDepth() error {
	l.subscribed.Lock()
	defer l.subscribed.Unlock()

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
	l.subscribed.Lock()
	defer l.subscribed.Unlock()

	if !l.subscribed.depth {
		return
	}
	msg := fmt.Sprintf(`{"method":"UNSUBSCRIBE","params":["%s@depth"],"id":1}`,
		strings.ToLower(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return
	}
	l.subscribed.depth = false
}

func (l *Listener) subscribeTrade() error {
	l.subscribed.Lock()
	defer l.subscribed.Unlock()

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
	l.subscribed.Lock()
	defer l.subscribed.Unlock()

	if !l.subscribed.trade {
		return
	}
	msg := fmt.Sprintf(`{"method":"UNSUBSCRIBE","params":["%s@trade"],"id":2}`,
		strings.ToLower(l.symbol))
	if err := l.sendWsMessage(msg); err != nil {
		return
	}
	l.subscribed.trade = false
}

func (l *Listener) fetchDepthSnapshot(ctx context.Context) {
	url := fmt.Sprintf("https://api.binance.com/api/v3/depth?symbol=%s&limit=1000",
		strings.ToUpper(l.symbol))

	req, err := http.NewRequestWithContext(l.ctx, "GET", url, nil)
	if err != nil {
		l.err(err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.err(err)
		return
	}
	received := timestamp.Stamp(time.Now())
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.err(err)
		return
	}

	if bytes.Contains(body, []byte("Illegal")) {
		l.err(errors.New(string(body)))
		return
	}
	if bytes.Contains(body, []byte("Invalid")) {
		l.err(errors.New(string(body)))
		return
	}

	var parser fastjson.Parser
	v, err := parser.ParseBytes(body)
	if err != nil {
		l.err(err)
		return
	}
	snapshot := l.parseDepthSnapshot(v)
	snapshot.Timestamp, snapshot.Received = received, received
	l.depth.snapshot.Store(snapshot)
}

func (l *Listener) process(msg []byte) error {
	received := timestamp.Stamp(time.Now())
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
			if ds := l.depth.snapshot.Load(); ds != nil && ds.(*DepthUpdateMessage) != nil {
				snapshot := ds.(*DepthUpdateMessage)
				l.sendDepthUpdate(snapshot)
				for _, du := range l.depth.updates {
					if du.FinalID < snapshot.FinalID+1 {
						continue
					}
					l.sendDepthUpdate(du)
				}
				l.depth.updates = nil
				l.depth.snapshot.Store((*DepthUpdateMessage)(nil))
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
		// fallthrough
	}
	if result := v.Get("result"); result != nil {
		return nil
	}
	if bytes.Contains(msg, []byte("error")) {
		return errors.New(string(msg))
	}
	l.warn(errors.New(string(msg)))
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
		l.warn(fmt.Errorf("missing depth updates %d:%d", l.depth.nextID, firstID))
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
		Timestamp: timestamp.Milli(v.GetInt64("E")),
		FirstID:   firstID,
		FinalID:   finalID,
		Bids:      bids,
		Asks:      asks,
	}
}

func (l *Listener) sendDepthUpdate(du *DepthUpdateMessage) {
	bookCh := l.bookCh.Load()
	if bookCh == nil || bookCh.(chan *exchange.BookUpdate) == nil {
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
		Timestamp:   timestamp.Milli(v.GetInt64("E")),
		Occurred:    timestamp.Milli(v.GetInt64("T")),
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
	if tradesCh == nil || tradesCh.(chan []*exchange.Trade) == nil {
		return
	}
	tradesCh.(chan []*exchange.Trade) <- []*exchange.Trade{
		{
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
	l.opts.Stderr.Printf("Stopping listener binance:%s", l.symbol)
	if bookCh := l.bookCh.Load(); bookCh != nil && bookCh.(chan *exchange.BookUpdate) != nil {
		l.unsubscribeDepth()
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
