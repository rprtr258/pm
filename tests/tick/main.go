package main

import (
	"context"
	"os"
	"time"

	flags "github.com/rprtr258/cli/contrib"
	"github.com/rprtr258/scuf"
)

type App struct {
	Interval time.Duration `long:"interval" short:"i" description:"interval between ticks, e.g. 100ms, 5s" default:"1s"`
}

func (x *App) Execute([]string) error {
	ctx := context.Background()

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
	if _, err := flags.NewParser(&struct {
		App App `command:"tick"` // TODO: unneeded subcommand, should be just root command
	}{}, flags.Default).ParseArgs(os.Args[1:]...); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Kind == flags.ErrHelp {
			return
		}
		os.Exit(1)
	}
}
