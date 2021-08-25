package binance

import (
	"sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

type DepthUpdateMessage struct {
	Timestamp timestamp.T
	Received  timestamp.T

	FirstID int64
	FinalID int64

	Bids []exchange.PriceLevelUpdate
	Asks []exchange.PriceLevelUpdate
}

type TradeMessage struct {
	Timestamp timestamp.T
	Received  timestamp.T
	Occurred  timestamp.T

	TradeID     int64
	BuyOrderID  int64
	SellOrderID int64

	Price    string
	Quantity string
	MakerBuy bool
}
