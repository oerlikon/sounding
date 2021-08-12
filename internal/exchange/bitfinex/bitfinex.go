package bitfinex

import (
	. "sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

const exchName = "Bitfinex"

type BookUpdateMessage struct {
	Timestamp Timestamp
	Received  Timestamp

	Bids []exchange.PriceLevelUpdate
	Asks []exchange.PriceLevelUpdate
}

type TradeMessage struct {
	Timestamp Timestamp
	Received  Timestamp
	Occurred  Timestamp

	TradeID int64

	Price    string
	Amount   string
	TakerBuy bool
}
