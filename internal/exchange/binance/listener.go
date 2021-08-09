package binance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	bookCh atomic.Value
	errsCh atomic.Value

	ws     *websocket.Conn
	parser fastjson.Parser

	depthUpdates  []*DepthUpdateMessage
	depthSnapshot atomic.Value
	depthStarted  bool

	nextDepthUpdateID int64
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
	if err := l.subscribeBook(); err != nil {
		l.err(err)
		return nil
	}
	go l.fetchDepthSnapshot()
	bookCh := make(chan *exchange.BookUpdate, 1)
	l.bookCh.Store(bookCh)
	return bookCh
}

func (l *Listener) Errs() <-chan error {
	if errsCh := l.errsCh.Load(); errsCh != nil {
		return errsCh.(chan error)
	}
	errsCh := make(chan error, 1)
	l.errsCh.Store(errsCh)
	return errsCh
}

func (l *Listener) err(err error) {
	if errsCh := l.errsCh.Load(); errsCh != nil {
		errsCh.(chan error) <- err
	} else if l.opts.Logger != nil {
		l.opts.Logger.Println("Error:", err)
	}
}

func (l *Listener) warn(err error) {
	if l.opts.Logger != nil {
		l.opts.Logger.Println("Warning:", err)
	}
}

func (l *Listener) subscribeBook() error {
	msg := bytes.Buffer{}
	msg.WriteString(fmt.Sprintf(
		"{\"method\":\"SUBSCRIBE\",\"params\":[\"%s@depth\"],\"id\":1}",
		strings.ToLower(l.symbol)))
	return l.ws.WriteMessage(websocket.TextMessage, msg.Bytes())
}

func (l *Listener) unsubscribeBook() {
	msg := bytes.Buffer{}
	msg.WriteString(fmt.Sprintf(
		"{\"method\":\"UNSUBSCRIBE\",\"params\":[\"%s@depth\"],\"id\":1}",
		strings.ToLower(l.symbol)))
	l.ws.WriteMessage(websocket.TextMessage, msg.Bytes())
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
	var parser fastjson.Parser
	v, err := parser.ParseBytes(body)
	if err != nil {
		l.err(err)
		return
	}
	ds := l.parseDepthSnapshot(v)
	ds.Timestamp, ds.Received = received, received
	l.depthSnapshot.Store(ds)
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

			if l.depthStarted {
				l.sendDepthUpdate(du)
				return nil
			}
			l.depthUpdates = append(l.depthUpdates, du)
			if depthSnapshot := l.depthSnapshot.Load(); depthSnapshot != nil {
				ds := depthSnapshot.(*DepthUpdateMessage)
				l.sendDepthUpdate(ds)
				for _, du := range l.depthUpdates {
					if du.FinalID < ds.FinalID+1 {
						continue
					}
					l.sendDepthUpdate(du)
				}
				l.depthUpdates, l.depthSnapshot = nil, atomic.Value{}
				l.depthStarted = true
			}
			return nil
		}
	}
	return nil
}

func (l *Listener) parseDepthSnapshot(v *fastjson.Value) *DepthUpdateMessage {
	lastUpdateID := v.GetInt64("lastUpdateId")

	var bids, asks []exchange.PriceLevelUpdate

	if b := v.GetArray("bids"); b != nil {
		bids = make([]exchange.PriceLevelUpdate, len(b))
		for i, pq := range b {
			bids[i].P = string(pq.GetStringBytes("0"))
			bids[i].Q = string(pq.GetStringBytes("1"))
		}
	}
	if a := v.GetArray("aaks"); a != nil {
		asks = make([]exchange.PriceLevelUpdate, len(a))
		for i, pq := range a {
			asks[i].P = string(pq.GetStringBytes("0"))
			asks[i].Q = string(pq.GetStringBytes("1"))
		}
	}

	if lastUpdateID == 0 || len(bids)+len(asks) == 0 {
		l.err(errors.New("zero depth snapshot?"))
		return nil
	}

	return &DepthUpdateMessage{
		FinalID: lastUpdateID,
		Bids:    bids,
		Asks:    asks,
	}
}

func (l *Listener) parseDepthUpdate(v *fastjson.Value) *DepthUpdateMessage {
	ts := Milli(v.GetInt64("E"))
	firstID, finalID := v.GetInt64("U"), v.GetInt64("u")
	if l.nextDepthUpdateID != 0 && l.nextDepthUpdateID != firstID {
		l.warn(fmt.Errorf("missing depth updates %d:%d",
			l.nextDepthUpdateID, firstID))
	}
	l.nextDepthUpdateID = finalID + 1

	var bids, asks []exchange.PriceLevelUpdate

	if b := v.GetArray("b"); b != nil {
		bids = make([]exchange.PriceLevelUpdate, len(b))
		for i, pq := range b {
			bids[i].P = string(pq.GetStringBytes("0"))
			bids[i].Q = string(pq.GetStringBytes("1"))
		}
	}
	if a := v.GetArray("a"); a != nil {
		asks = make([]exchange.PriceLevelUpdate, len(a))
		for i, pq := range a {
			asks[i].P = string(pq.GetStringBytes("0"))
			asks[i].Q = string(pq.GetStringBytes("1"))
		}
	}

	return &DepthUpdateMessage{
		Timestamp: ts,
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

func (l *Listener) shutdown() {
	if l.opts.Logger != nil {
		l.opts.Logger.Printf("Stopping listener binance:%s", l.symbol)
	}
	if bookCh := l.bookCh.Load(); bookCh != nil {
		l.unsubscribeBook()
		close(bookCh.(chan *exchange.BookUpdate))
		l.bookCh = atomic.Value{}
	}
	l.ws.Close()
}
