package exchange

import (
	"context"

	. "sounding/internal/common/timestamp"
)

type Listener interface {
	Exchange() string
	Symbol() string

	Start(ctx context.Context) error

	Book() <-chan *BookUpdate
	Errs() <-chan error
}

type Side int

const (
	Bid Side = 1
	Buy Side = Bid

	Ask  Side = 2
	Sell Side = Ask
)

type BookUpdate struct {
	Exchange string
	Symbol   string

	Timestamp Timestamp
	Received  Timestamp

	Bids []PriceLevelUpdate
	Asks []PriceLevelUpdate
}

type PriceLevelUpdate struct {
	P string
	Q string
}
