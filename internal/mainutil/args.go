package mainutil

import (
	"os"
	"strings"
	"time"

	"github.com/mattn/go-shellwords"
	flag "github.com/spf13/pflag"
)

func ParseArgs(flags *flag.FlagSet) (argv []string, err error) {
	var argx []string
	if input, err := ReadAllStdin(); err == nil && len(input) > 0 {
		parser := shellwords.NewParser()
		parser.ParseEnv = true
		words, err := parser.Parse(b2s(input))
		if err != nil {
			return nil, err
		}
		argx = words
	} else if err != nil {
		return nil, err
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	argv = append([]string{}, flags.Args()...)
	return argv, flags.Parse(append(os.Args[1:], argx...))
}

func ParseTime(s string) (t time.Time, err error) {
	if s == "" || s == "-" || s == "0" {
		return time.Time{}, nil
	}
	if strings.ContainsAny(s, "T_ ") {
		s = strings.NewReplacer("T", " ", "_", " ", "   ", " ", "  ", " ").Replace(s)
	}
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999",
	}
	for i := len(formats) - 1; i > 0; i-- {
		if t, err := time.Parse(formats[i], s); err == nil {
			return t, nil
		}
	}
	return time.Parse(formats[0], s)
}
