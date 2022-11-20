package kraken

import (
	"github.com/oerlikon/sounding/internal/common/timestamp"
	"github.com/oerlikon/sounding/internal/exchange"
)

type BookUpdateMessage struct {
	Timestamp timestamp.T
	Received  timestamp.T

	Bids []exchange.PriceLevelUpdate
	Asks []exchange.PriceLevelUpdate
}

type TradeMessage struct {
	Timestamp timestamp.T
	Received  timestamp.T
	Occurred  timestamp.T

	Price  string
	Volume string
	Taker  exchange.Side
}
