package binance

import (
	"context"

	"sounding/internal/exchange"
)

type Listener struct {
	symbol string
	opts   Options
}

func NewListener(symbol string, options ...exchange.Option) exchange.Listener {
	var opts Options
	for _, o := range options {
		err := o(&opts)
		if err == exchange.ErrCommonOption {
			err = o(&opts.Options)
		}
		if err != nil {
			panic("unknown error setting options")
		}
	}
	return &Listener{
		symbol: symbol,
		opts:   opts,
	}
}

func (l *Listener) Symbol() string {
	return l.symbol
}

func (l *Listener) Start(ctx context.Context) error {
	return nil
}

func (l *Listener) Book() chan<- exchange.BookUpdate {
	return nil
}
