package exchange

import (
	"errors"

	stdlog "log"
)

type Options struct {
	Logger *stdlog.Logger
}

type Option func(opts interface{}) error

var ErrCommonOption = errors.New("common option")

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
