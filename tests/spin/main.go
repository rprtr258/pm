// Test program that pegs a core and ignores interrupts.

package main

import (
	"os"
	"os/signal"
)

func main() {
	signal.Ignore(os.Interrupt, os.Interrupt)
	for { //nolint:revive,staticcheck // of course it will use 100% of cpu
		// waste cpu here
	}
}
