package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	flag "github.com/spf13/pflag"

	. "sounding/internal/common"
	"sounding/internal/mainutil"
)

var Options struct {
	CPUProfile string
	Help       bool
}

var flags flag.FlagSet

func init() {
	flags.StringVarP(&Options.CPUProfile, "cpuprofile", "", "", "cpu profile")
	flags.BoolVarP(&Options.Help, "help", "", false, "this help message")
	flags.SetInterspersed(false)
	flags.SetOutput(io.Discard)
}

var exchanges = []string{"bitfinex", "kraken"}

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
		n := strings.IndexByte(arg, ':')
		if n < 1 || n > len(arg)-2 {
			return fmt.Errorf("invalid arg: %s", arg), 1
		}
		exch, sym := arg[:n], arg[n+1:]
		if FindString(exchanges, exch) < 0 {
			return fmt.Errorf("unknown exchange: %s", exch), 1
		}
		if symbols[exch] != "" && symbols[exch] != sym {
			return fmt.Errorf("more than one symbol for %s: %s", exch, sym), 1
		}
		symbols[exch] = sym
	}

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
