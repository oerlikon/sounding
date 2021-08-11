package binance

import (
	. "sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

const exchName = "Binance"

type DepthUpdateMessage struct {
	Timestamp Timestamp
	Received  Timestamp

	FirstID int64
	FinalID int64

	Bids []exchange.PriceLevelUpdate
	Asks []exchange.PriceLevelUpdate
}

type TradeMessage struct {
	Timestamp Timestamp
	Received  Timestamp
	Occurred  Timestamp

	TradeID     int64
	BuyOrderID  int64
	SellOrderID int64

	Price    string
	Quantity string
	MakerBuy bool
}
