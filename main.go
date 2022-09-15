package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/urfave/cli/v2"
)

// processes - proc name -> pid
var (
	processes map[string]int = make(map[string]int)
	userHome                 = os.Getenv("HOME")
	homeDir                  = path.Join(userHome, ".pm")
)

func main() {
	var name string
	runCmd := &cli.Command{
		Name:      "start",
		Aliases:   []string{"run", "r"},
		ArgsUsage: "<cmd> [args...]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Aliases:     []string{"n"},
				Required:    true,
				Destination: &name,
			},
		},
		Action: func(ctx *cli.Context) error {
			args := ctx.Args().Slice()
			if len(args) < 1 {
				return errors.New("Command expected")
			}

			if err := os.Mkdir(path.Join(homeDir, name), 0755); err != nil {
				return err
			}

			stdoutLogFile, err := os.OpenFile(path.Join(homeDir, name, "stdout"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}

			stderrLogFile, err := os.OpenFile(path.Join(homeDir, name, "stderr"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}

			cmd := exec.CommandContext(ctx.Context, args[0], args[1:]...)
			cmd.Stdout = stdoutLogFile
			cmd.Stderr = stderrLogFile
			if err := cmd.Start(); err != nil {
				return err
			}

			processes[name] = cmd.Process.Pid

			return nil
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
		Before: func(*cli.Context) error {
			if _, err := os.Stat(homeDir); os.IsNotExist(err) {
				os.Mkdir(homeDir, 0755)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
