// Test program that pegs a core and ignores interrupts.

package main

import (
	"os/signal"
	"syscall"
)

func main() {
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM)
	for { //nolint:revive,staticcheck // of course it will use 100% of cpu
		// waste cpu here
	}
}
