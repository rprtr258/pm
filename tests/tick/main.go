package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rprtr258/log"
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

			i := 0
			for {
				i++
				fmt.Printf("tick %d at %v\n", i, time.Now())
				<-ticker.C
			}
		},
	}).Run(os.Args); err != nil {
		log.Fatal(err.Error())
	}
}
