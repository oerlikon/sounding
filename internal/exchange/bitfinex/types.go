package bitfinex

import (
	"sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

type BookUpdateMessage struct {
	Timestamp timestamp.Timestamp
	Received  timestamp.Timestamp

	Bids []exchange.PriceLevelUpdate
	Asks []exchange.PriceLevelUpdate
}

type TradeMessage struct {
	Timestamp timestamp.Timestamp
	Received  timestamp.Timestamp
	Occurred  timestamp.Timestamp

	TradeID int64

	Price    string
	Amount   string
	TakerBuy bool
}
