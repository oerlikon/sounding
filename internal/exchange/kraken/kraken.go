package kraken

import (
	"sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

const exchName = "Kraken"

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

	Price  string
	Volume string
	Taker  exchange.Side
}
