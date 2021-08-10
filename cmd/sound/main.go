package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"

	flag "github.com/spf13/pflag"
	"golang.org/x/term"

	. "sounding/internal/common"
	"sounding/internal/exchange"
	"sounding/internal/exchange/binance"
	"sounding/internal/mainutil"
)

var Options struct {
	Books      bool
	Trades     bool
	ExpID      int
	CPUProfile string
	Help       bool
}

var flags flag.FlagSet

func init() {
	flags.BoolVarP(&Options.Books, "books", "B", true, "books")
	flags.BoolVarP(&Options.Trades, "trades", "T", true, "trades")
	flags.IntVarP(&Options.ExpID, "id", "", 0, "experiment ID")
	flags.StringVarP(&Options.CPUProfile, "cpuprofile", "", "", "cpu profile")
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
			if !FindString(exchanges, exch) {
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

	listeners := make([]exchange.Listener, 0, len(exchanges))
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
			listeners = append(listeners, binance.NewListener(symbol,
				exchange.OptionLogger(stderr)))
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

	writer := bufio.NewWriterSize(os.Stdout, 1*MiB)
	wg := sync.WaitGroup{}

	if Options.Books {
		wg.Add(1)
		go BooksLoop(listeners, writer, &wg)
	}
	if Options.Trades {
		wg.Add(1)
		go TradesLoop(listeners, writer, &wg)
	}

	wg.Wait()
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

func BooksLoop(listeners []exchange.Listener, w io.StringWriter, wg *sync.WaitGroup) {
	exp := ""
	if Options.ExpID != 0 {
		exp = fmt.Sprintf("%d,", Options.ExpID)
	}
	cases := make([]reflect.SelectCase, 0, len(listeners))
	for _, listener := range listeners {
		if listener == nil {
			continue
		}
		if book := listener.Book(); book != nil {
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(book),
			})
		}
	}
	for len(cases) > 0 {
		n, value, ok := reflect.Select(cases)
		if !ok {
			cases = append(cases[:n], cases[n+1:]...)
			continue
		}
		bu := value.Interface().(*exchange.BookUpdate)
		for _, pl := range bu.Bids {
			w.WriteString(fmt.Sprintf("B %s%d,%s,%s,%s,%s,%s,%s\n",
				exp,
				bu.Timestamp.UnixMilli(),
				bu.Timestamp.Format("2006-01-02 15:04:05.000"),
				bu.Exchange,
				strings.ToUpper(bu.Symbol),
				"BID",
				pl.Price,
				pl.Quantity))
		}
		for _, pl := range bu.Asks {
			w.WriteString(fmt.Sprintf("B %s%d,%s,%s,%s,%s,%s,%s\n",
				exp,
				bu.Timestamp.UnixMilli(),
				bu.Timestamp.Format("2006-01-02 15:04:05.000"),
				bu.Exchange,
				strings.ToUpper(bu.Symbol),
				"ASK",
				pl.Price,
				pl.Quantity))
		}
	}
	wg.Done()
}

func TradesLoop(listeners []exchange.Listener, w io.StringWriter, wg *sync.WaitGroup) {
	exp := ""
	if Options.ExpID != 0 {
		exp = fmt.Sprintf("%d,", Options.ExpID)
	}
	cases := make([]reflect.SelectCase, 0, len(listeners))
	for _, listener := range listeners {
		if listener == nil {
			continue
		}
		if trades := listener.Trades(); trades != nil {
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(trades),
			})
		}
	}
	for len(cases) > 0 {
		n, value, ok := reflect.Select(cases)
		if !ok {
			cases = append(cases[:n], cases[n+1:]...)
			continue
		}
		trade := value.Interface().(*exchange.Trade)
		w.WriteString(fmt.Sprintf("T %s%d,%s,%s,%s,%s,%s,%s\n",
			exp,
			trade.Timestamp.UnixMilli(),
			trade.Timestamp.Format("2006-01-02 15:04:05.000"),
			trade.Exchange,
			strings.ToUpper(trade.Symbol),
			func() string {
				if trade.Maker == exchange.Bid {
					return "BID"
				}
				return "ASK"
			}(),
			trade.Price,
			trade.Quantity))
	}
	wg.Done()
}
