package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	runCmd := &cli.Command{
		Name:    "start",
		Aliases: []string{"run", "r"},
		Action: func(ctx *cli.Context) error {
			fmt.Println(ctx.Args().Slice())
			return nil
		},
	}
	stopCmd := &cli.Command{
		Name:    "stop",
		Aliases: []string{"kill", "s"},
		Action: func(ctx *cli.Context) error {
			fmt.Println(ctx.Args().Slice())
			return nil
		},
	}
	app := &cli.App{
		Name:  "pm",
		Usage: "manage running processes",
		Commands: []*cli.Command{
			runCmd,
			stopCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err.Error())
	}
}
