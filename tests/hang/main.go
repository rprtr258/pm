// Test program that blocks forever and ignores interrupts.

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func printSignals() {
	sigCh := make(chan os.Signal, 100)
	signal.Notify(sigCh, os.Interrupt, os.Interrupt)
	for sig := range sigCh {
		log.Info().
			Int("sig", int(sig.(syscall.Signal))).
			Stringer("signal", sig).
			Msg("received signal")
	}
}

func main() {
	go printSignals()
	time.Sleep(24 * 365 * 100 * time.Hour)
}
