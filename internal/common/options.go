package common

import (
	"errors"
	"fmt"

	"github.com/oerlikon/structs"
	"github.com/rs/zerolog"
)

type Option func(options interface{}) error

var ErrBadOption = errors.New("bad option")

func OptionStdout(stdout Printlnfer) Option {
	return func(options interface{}) error {
		s := structs.New(options)
		field := s.Field("Stdout")
		if field == nil {
			return ErrBadOption
		}
		if err := field.Set(stdout); err != nil {
			return fmt.Errorf("%w: %s", ErrBadOption, err)
		}
		return nil
	}
}

func OptionStderr(stderr Printlnfer) Option {
	return func(options interface{}) error {
		s := structs.New(options)
		field := s.Field("Stderr")
		if field == nil {
			return ErrBadOption
		}
		if err := field.Set(stderr); err != nil {
			return fmt.Errorf("%w: %s", ErrBadOption, err)
		}
		return nil
	}
}

func OptionLogger(logger zerolog.Logger) Option {
	return func(options interface{}) error {
		s := structs.New(options)
		field := s.Field("Logger")
		if field == nil {
			return ErrBadOption
		}
		if err := field.Set(logger); err != nil {
			return fmt.Errorf("%w: %s", ErrBadOption, err)
		}
		return nil
	}
}
