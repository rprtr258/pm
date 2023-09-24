package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rprtr258/pm/internal/infra/cli/log/buffer"
	"github.com/urfave/cli/v2"
)

func main() {
	if err := (&cli.App{
		Name: "tick",
		Flags: []cli.Flag{&cli.DurationFlag{
			Name:    "interval",
			Aliases: []string{"i"},
			Usage:   "interval between ticks, e.g. 100ms, 5s",
			Value:   time.Second,
		}},
		Action: func(ctx *cli.Context) error {
			ticker := time.NewTicker(ctx.Duration("interval"))
			defer ticker.Stop()

			for i := 0; ; i++ {
				select {
				case <-ctx.Context.Done():
					return nil
				case now := <-ticker.C:
					fmt.Println(buffer.NewString(func(b *buffer.Buffer) {
						b.
							String(now.Format(time.RFC3339), buffer.ColorFaint).
							String(": tick").
							Styled(func(b *buffer.Buffer) {
								b.Printf("%4d", i)
							}, buffer.FgBlue)
					}))
				}
			}
		},
	}).Run(os.Args); err != nil {
		log.Fatal(err.Error())
	}
}
