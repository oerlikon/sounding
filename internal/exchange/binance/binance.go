package binance

import (
	. "sounding/internal/common/timestamp"
	"sounding/internal/exchange"
)

const exchName = "Binance"

type DepthUpdateMessage struct {
	Timestamp Timestamp
	Received  Timestamp
	FirstID   int64
	FinalID   int64
	Bids      []exchange.PriceLevelUpdate
	Asks      []exchange.PriceLevelUpdate
}

type DepthSnapshotMessage struct {
	Received     Timestamp
	LastUpdateID int64
	Bids         []exchange.PriceLevelUpdate
	Asks         []exchange.PriceLevelUpdate
}
