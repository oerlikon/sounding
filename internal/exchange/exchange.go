package exchange

import . "sounding/internal/common/timestamp"

type Listener interface {
	Symbol() string
	Book() chan<- BookUpdate
}

type BookUpdate struct {
	T    Timestamp
	P, V float64
}
