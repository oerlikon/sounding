package mainutil

import (
	"fmt"
	"os"
	"time"

	bar "github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

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
