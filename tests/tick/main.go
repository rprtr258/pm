package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rprtr258/scuf"
	"github.com/rs/zerolog/log"
)

func run(ctx context.Context, intervalStr string) error {
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("parse interval %q: %w", intervalStr, err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			scuf.New(os.Stdout).
				String(now.Format(time.RFC3339), scuf.ModFaint).
				String(": ick").
				Styled(func(b scuf.Buffer) {
					b.Printf("%4d", i)
				}, scuf.FgBlue).
				NL()
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "1s")
	}

	if err := run(context.Background(), os.Args[1]); err != nil {
		log.Fatal().Err(err).Send()
	}
}
