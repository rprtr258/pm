package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rprtr258/scuf"
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
					fmt.Println(scuf.NewString(func(b scuf.Buffer) {
						b.
							String(now.Format(time.RFC3339), scuf.ModFaint).
							String(": tick").
							Styled(func(b scuf.Buffer) {
								b.Printf("%4d", i)
							}, scuf.FgBlue)
					}))
				}
			}
		},
	}).Run(os.Args); err != nil {
		log.Fatal(err.Error())
	}
}
