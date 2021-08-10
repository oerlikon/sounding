package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	flag "github.com/spf13/pflag"
	"golang.org/x/term"

	. "sounding/internal/common"
	"sounding/internal/exchange"
	"sounding/internal/exchange/binance"
	"sounding/internal/mainutil"
)

var Options struct {
	CPUProfile string
	ExpID      int
	Help       bool
}

var flags flag.FlagSet

func init() {
	flags.StringVarP(&Options.CPUProfile, "cpuprofile", "", "", "cpu profile")
	flags.IntVarP(&Options.ExpID, "id", "", 0, "experiment ID")
	flags.BoolVarP(&Options.Help, "help", "", false, "this help message")
	flags.SetInterspersed(false)
	flags.SetOutput(io.Discard)
}

var exchanges = []string{"binance", "bitfinex", "huobi", "kraken"}

func run() (err error, ret int) {
	if err := mainutil.ParseArgs(&flags); err != nil {
		if err == flag.ErrHelp {
			Options.Help = true
		} else {
			return err, 1
		}
	}
	if Options.Help {
		stdout.Print(flags.FlagUsages())
		return nil, 1
	}
	if err := mainutil.Validate(Options); err != nil {
		stderr.Print(err)
		return nil, 1
	}
	if flags.NArg() == 0 {
		stderr.Print("Symbols?")
		return nil, 1
	}

	symbols := map[string]string{}
	for _, arg := range flags.Args() {
		if n := strings.IndexByte(arg, ':'); n >= 0 {
			exch, sym := arg[:n], arg[n+1:]
			if exch == "" || sym == "" {
				return fmt.Errorf("bad arg: %s", arg), 1
			}
			if !ContainsString(exchanges, exch) {
				return fmt.Errorf("unknown exchange: %s", exch), 1
			}
			if symbols[exch] != "" && symbols[exch] != sym {
				return fmt.Errorf("more than one symbol for %s: %s", exch, sym), 1
			}
			symbols[exch] = sym
		} else {
			if symbols["*"] != "" && symbols["*"] != arg {
				return fmt.Errorf("more than one symbol for all exchanges: %s", arg), 1
			}
			symbols["*"] = arg
		}
	}

	listeners := make(map[string]exchange.Listener, len(exchanges))
	for _, exch := range exchanges {
		symbol, ok := symbols[exch]
		if !ok {
			symbol = symbols["*"]
		}
		if symbol == "" {
			return fmt.Errorf("no symbol for exchange: %s", exch), 1
		}
		switch exch {
		case "binance":
			listeners[exch] = binance.NewListener(symbol, exchange.OptionLogger(stderr))
		case "bitfinex":
		case "huobi":
		case "kraken":
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		if term.IsTerminal(int(os.Stderr.Fd())) {
			fmt.Fprintf(os.Stderr, "\r  \r") // Erase possible ^C.
		}
		cancel()
	}()

	for _, listener := range listeners {
		if err := listener.Start(ctx); err != nil {
			return err, 2
		}
	}

	expID := ""
	if Options.ExpID != 0 {
		expID = fmt.Sprintf("%d,", Options.ExpID)
	}

	writer := bufio.NewWriterSize(os.Stdout, 1*MiB)
	for du := range listeners["binance"].Book() {
		for _, pl := range du.Bids {
			writer.WriteString(fmt.Sprintf("U %s%d,%s,%s,%s,%s,%s,%s\n",
				expID,
				du.Timestamp.UnixMilli(),
				du.Timestamp.Format("2006-01-02 15:04:05.000"),
				du.Exchange,
				strings.ToUpper(du.Symbol),
				"BID",
				pl.P,
				pl.Q))
		}
		for _, pl := range du.Asks {
			writer.WriteString(fmt.Sprintf("U %s%d,%s,%s,%s,%s,%s,%s\n",
				expID,
				du.Timestamp.UnixMilli(),
				du.Timestamp.Format("2006-01-02 15:04:05.000"),
				du.Exchange,
				strings.ToUpper(du.Symbol),
				"ASK",
				pl.P,
				pl.Q))
		}
	}
	writer.Flush()

	return nil, 0
}

func main() {
	err, ret := run()
	if err != nil {
		stderr.Println("Error:", err)
	}
	if ret != 0 {
		os.Exit(ret)
	}
}
