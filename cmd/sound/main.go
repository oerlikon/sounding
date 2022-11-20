package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"

	flag "github.com/spf13/pflag"
	"golang.org/x/term"

	. "sounding/internal/common"
	"sounding/internal/common/syncio"
	"sounding/internal/exchange"
	"sounding/internal/exchange/binance"
	"sounding/internal/exchange/bitfinex"
	"sounding/internal/exchange/kraken"
	"sounding/internal/mainutil"
)

var Options struct {
	Books      bool
	Trades     bool
	CPUProfile string
	Help       bool
}

var flags flag.FlagSet

func init() {
	flags.BoolVarP(&Options.Books, "books", "B", true, "books")
	flags.BoolVarP(&Options.Trades, "trades", "T", true, "trades")
	flags.StringVarP(&Options.CPUProfile, "cpuprofile", "", "", "cpu profile")
	flags.BoolVarP(&Options.Help, "help", "", false, "this help message")
	flags.SetInterspersed(false)
	flags.SetOutput(io.Discard)
}

var exchanges = []string{"binance", "bitfinex", "kraken"}

func run() (int, error) {
	if _, err := mainutil.ParseArgs(&flags); err != nil {
		if err == flag.ErrHelp {
			Options.Help = true
		} else {
			return 1, err
		}
	}
	if Options.Help {
		stdout.Print(flags.FlagUsages())
		return 1, nil
	}
	if err := mainutil.Validate(Options); err != nil {
		stderr.Print(err)
		return 1, nil
	}
	if flags.NArg() == 0 {
		stderr.Print("Symbols?")
		return 1, nil
	}

	symbols := map[string]string{}
	for _, arg := range flags.Args() {
		n := strings.IndexByte(arg, ':')
		if n < 1 || n > len(arg)-2 {
			return 1, fmt.Errorf("invalid arg: %s", arg)
		}
		exch, sym := arg[:n], arg[n+1:]
		if FindString(exchanges, exch) < 0 {
			return 1, fmt.Errorf("unknown exchange: %s", exch)
		}
		if symbols[exch] != "" && symbols[exch] != sym {
			return 1, fmt.Errorf("more than one symbol for %s: %s", exch, sym)
		}
		symbols[exch] = sym
	}

	listeners := make([]exchange.Listener, 0, len(exchanges))
	for _, exch := range exchanges {
		symbol := symbols[exch]
		if symbol == "" {
			continue
		}
		switch exch {
		case "binance":
			listeners = append(listeners, binance.NewListener(symbol, OptionStderr(stderr)))
		case "bitfinex":
			listeners = append(listeners, bitfinex.NewListener(symbol, OptionStderr(stderr)))
		case "kraken":
			listeners = append(listeners, kraken.NewListener(symbol, OptionStderr(stderr)))
		}
	}
	if len(listeners) == 0 {
		stderr.Print("No listeners?")
		return 1, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex // Initialization mutex.
	mu.Lock()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		if term.IsTerminal(int(os.Stderr.Fd())) {
			fmt.Fprintf(os.Stderr, "\r  \r") // Erase possible ^C.
		}
		mu.Lock()
		defer mu.Unlock()
		cancel()
	}()

	for _, listener := range listeners {
		if err := listener.Start(ctx); err != nil {
			return 2, err
		}
	}

	out := bufio.NewWriterSize(os.Stdout, 1*MiB)
	writer := syncio.NewStringWriter(out)
	wg := sync.WaitGroup{}

	if Options.Books {
		wg.Add(1)
		go BooksLoop(Books(listeners), writer, &wg)
	}
	if Options.Trades {
		wg.Add(1)
		go TradesLoop(Trades(listeners), writer, &wg)
	}

	stderr.Print("Listening...")
	mu.Unlock()

	wg.Wait()
	out.Flush()

	return 0, nil
}

func main() {
	ret, err := run()
	if err != nil {
		stderr.Println("Error:", err)
	}
	if ret != 0 {
		os.Exit(ret)
	}
}
