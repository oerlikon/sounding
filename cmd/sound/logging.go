package main

import (
	stdlog "log"
	"os"
)

var (
	stdout = stdlog.New(os.Stdout, "", 0)
	stderr = stdlog.New(os.Stderr, "", stdlog.Ltime|stdlog.Lmicroseconds)
)
