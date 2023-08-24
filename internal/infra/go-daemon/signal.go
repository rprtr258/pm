package daemon

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

// ErrStop should be returned signal handler function
// for termination of handling signals.
var ErrStop = errors.New("stop serve signals")

type SignalHandlerFunc func(sig os.Signal) error

// SetSigHandler sets handler for the given signals.
// SIGTERM has the default handler, which returns ErrStop.
func SetSigHandler(handler SignalHandlerFunc, signals ...os.Signal) {
	for _, sig := range signals {
		handlers[sig] = handler
	}
}

// ServeSignals calls handlers for system signals.
func ServeSignals() error {
	sigsCh := make(chan os.Signal, len(handlers))
	for sig := range handlers {
		signal.Notify(sigsCh, sig)
	}

	var err error
	for sig := range sigsCh {
		err = handlers[sig](sig)
		if err != nil {
			break
		}
	}

	signal.Stop(sigsCh)

	if err == ErrStop {
		return nil
	}

	return err
}

var handlers = map[os.Signal]SignalHandlerFunc{
	syscall.SIGTERM: func(sig os.Signal) error {
		log.Info().Msg("SIGTERM received")
		return ErrStop
	},
}
