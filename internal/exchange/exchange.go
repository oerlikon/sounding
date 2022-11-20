package exchange

import (
	"context"

	"github.com/oerlikon/sounding/internal/common/timestamp"
)

//
// Listener

type Listener interface {
	Exchange() string
	Symbol() string

	Start(ctx context.Context) error

	Book() <-chan *BookUpdate
	Trades() <-chan []*Trade
}

//
// Sides

type Side int

const (
	Bid Side = 1
	Buy Side = Bid

	Ask  Side = 2
	Sell Side = Ask
)

//
// Book

type BookUpdate struct {
	Exchange string
	Symbol   string

	Timestamp timestamp.T
	Received  timestamp.T

	Bids []PriceLevelUpdate
	Asks []PriceLevelUpdate
}

type PriceLevelUpdate struct {
	Price    string
	Quantity string
}

//
// Trade

type Trade struct {
	Exchange string
	Symbol   string

	Timestamp timestamp.T
	Received  timestamp.T
	Occurred  timestamp.T

	TradeID     int64
	BuyOrderID  int64
	SellOrderID int64

	Price    string
	Quantity string
	Taker    Side
}
