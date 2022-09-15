package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"
)

func main() {
	runCmd := &cli.Command{
		Name:      "start",
		Aliases:   []string{"run", "r"},
		ArgsUsage: "<cmd> [args...]",

		Action: func(ctx *cli.Context) error {
			args := ctx.Args().Slice()
			if len(args) < 1 {
				return errors.New("Command expected")
			}

			cmd := exec.CommandContext(ctx.Context, args[0], args[1:]...)
			cmd.Stdout = os.Stdout
			err := cmd.Run()
			return err
		},
	}
	stopCmd := &cli.Command{
		Name:      "stop",
		Aliases:   []string{"kill", "s"},
		ArgsUsage: "<name>",
		Action: func(ctx *cli.Context) error {
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
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
