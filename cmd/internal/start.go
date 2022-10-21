package internal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rprtr258/pm/api"
	"github.com/urfave/cli/v2"
)

const (
	_flagName = "name"
)

func init() {
	AllCmds = append(AllCmds, StartCmd)
}

var StartCmd = &cli.Command{
	Name: "start",
	// ArgsUsage: "<'cmd args...'|name|namespace|config|id>...",
	ArgsUsage: "cmd args...",
	Usage:     "start and daemonize an app",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    _flagName,
			Aliases: []string{"n"},
			Usage:   "set a name for the process",
		},
		// &cli.BoolFlag{Name: "watch", Usage: "Watch folder for changes"},
		// &cli.BoolFlag{Name: "fresh", Usage: "Rebuild Dockerfile"},
		// &cli.BoolFlag{Name: "daemon", Usage: "Run container in Daemon mode (debug purposes)"},
		// &cli.BoolFlag{Name: "container", Usage: "Start application in container mode"},
		// &cli.BoolFlag{Name: "dist", Usage: "with --container; change local Dockerfile to containerize all files in current directory"},
		// &cli.StringFlag{Name: "image-name", Usage: "with --dist; set the exported image name"},
		// &cli.BoolFlag{Name: "node-version", Usage: "with --container, set a specific major Node.js version"},
		// &cli.BoolFlag{Name: "dockerdaemon", Usage: "for debugging purpose"},
	},
	Action: func(ctx *cli.Context) error {
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferFunc()

		name := ctx.String(_flagName)

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("command expected")
		}

		resp, err := client.Start(ctx.Context, &api.StartReq{
			Name: name,
			Cwd:  ".",
			Tags: &api.Tags{Tags: []string{}},
			Cmd:  strings.Join(args, " "),
		})
		if err != nil {
			return err
		}

		fmt.Printf("got: id=%d pid=%d", resp.GetId(), resp.GetPid())
		return nil
	},
}
