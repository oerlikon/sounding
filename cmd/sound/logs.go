package main

import (
	"log"
	"os"
)

var (
	stdout = log.New(os.Stdout, "", 0)
	stderr = log.New(os.Stderr, "", log.Ltime|log.Lmicroseconds)
)
