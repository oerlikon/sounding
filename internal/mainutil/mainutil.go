package mainutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-shellwords"
	bar "github.com/schollz/progressbar/v3"
	flag "github.com/spf13/pflag"
	"golang.org/x/term"
)

func ReadAllStdin() ([]byte, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}
	return io.ReadAll(os.Stdin)
}

func ParseArgs(flags *flag.FlagSet) error {
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	narg := flags.NArg()

	var argx []string

	input, err := ReadAllStdin()
	if err != nil {
		return err
	}
	if len(input) > 0 {
		parser := shellwords.NewParser()
		parser.ParseEnv = true
		words, err := parser.Parse(string(input))
		if err != nil {
			return err
		}
		if err := flags.Parse(words); err != nil {
			return err
		}
		if narg != 0 && flags.NArg() != 0 {
			return errors.New("non-option args on both inputs")
		}
		argx = words
	}

	argv := os.Args[1:]
	if len(argx) > 0 {
		if narg != 0 || flags.NArg() == 0 {
			argv = append(argx, argv...)
		} else {
			argv = append(argv, argx...)
		}
	}
	return flags.Parse(argv)
}

func ParseTimeArg(s string) (t time.Time, err error) {
	if s == "" || s == "-" || s == "0" {
		return time.Time{}, nil
	}
	if strings.IndexAny(s, "T_ ") != -1 {
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

func NewProgressBar(count int, options ...bar.Option) *bar.ProgressBar {
	return bar.NewOptions(count,
		append([]bar.Option{
			bar.OptionSetDescription(""),
			bar.OptionSetWriter(os.Stderr),
			bar.OptionSetVisibility(term.IsTerminal(int(os.Stderr.Fd()))),
			bar.OptionSetWidth(33),
			bar.OptionThrottle(99 * time.Millisecond),
			bar.OptionSetTheme(bar.Theme{
				Saucer:        "#",
				SaucerPadding: ".",
				BarStart:      "[",
				BarEnd:        "]",
			}),
			bar.OptionSpinnerType(9),
			bar.OptionShowCount(),
			bar.OptionSetRenderBlankState(true),
			bar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
		}, options...)...)
}
