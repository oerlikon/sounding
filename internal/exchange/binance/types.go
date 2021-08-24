package binance

import (
	"sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

type DepthUpdateMessage struct {
	Timestamp timestamp.Timestamp
	Received  timestamp.Timestamp

	FirstID int64
	FinalID int64

	Bids []exchange.PriceLevelUpdate
	Asks []exchange.PriceLevelUpdate
}

type TradeMessage struct {
	Timestamp timestamp.Timestamp
	Received  timestamp.Timestamp
	Occurred  timestamp.Timestamp

	TradeID     int64
	BuyOrderID  int64
	SellOrderID int64

	Price    string
	Quantity string
	MakerBuy bool
}
