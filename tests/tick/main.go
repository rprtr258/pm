package main

import (
	"context"
	"os"
	"time"

	"github.com/rprtr258/cli"
	"github.com/rprtr258/scuf"
	"github.com/rs/zerolog/log"
)

type app struct {
	Interval time.Duration `long:"interval" short:"i" description:"interval between ticks, e.g. 100ms, 5s" default:"1s"`
}

func (x app) Execute(ctx context.Context) error {
	ticker := time.NewTicker(x.Interval)
	defer ticker.Stop()

	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			scuf.New(os.Stdout).
				String(now.Format(time.RFC3339), scuf.ModFaint).
				String(": tick").
				Styled(func(b scuf.Buffer) {
					b.Printf("%4d", i)
				}, scuf.FgBlue).
				NL()
		}
	}
}

func main() {
	if err := cli.RunContext[app](context.Background(), os.Args...); err != nil {
		log.Fatal().Err(err).Send()
	}
}
