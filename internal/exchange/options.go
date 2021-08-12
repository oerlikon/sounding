package exchange

import (
	"errors"

	stdlog "log"
)

var ErrCommonOption = errors.New("exchange: common option")

type Options struct {
	Logger *stdlog.Logger
}

type Option func(opts interface{}) error

func OptionLogger(log *stdlog.Logger) Option {
	return func(opts interface{}) error {
		options, ok := opts.(*Options)
		if !ok {
			return ErrCommonOption
		}
		options.Logger = log
		return nil
	}
}
