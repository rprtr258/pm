package internal

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/rprtr258/pm/api"
	"github.com/urfave/cli/v2"
)

func init() {
	AllCmds = append(AllCmds, StopCmd)
}

var StopCmd = &cli.Command{
	Name:      "stop",
	Usage:     "stop a process",
	ArgsUsage: "<id|name|namespace|all|json>...",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "watch",
			Usage: "stop watching for file changes",
		},
		&cli.BoolFlag{
			Name:  "kill",
			Usage: "kill process with SIGKILL instead of SIGINT",
		},
		&cli.DurationFlag{
			Name:    "kill-timeout",
			Aliases: []string{"k"},
			Usage:   "delay before sending final SIGKILL signal to process",
		},
		&cli.BoolFlag{
			Name:  "no-treekill",
			Usage: "Only kill the main process, not detached children",
		},
		// TODO: -i/... to confirm which procs will be stopped
	},
	Action: func(ctx *cli.Context) error {
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferErr(deferFunc)

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("what to stop expected")
		}

		ids := make([]uint64, len(args))
		for i, arg := range args {
			ids[i], err = strconv.ParseUint(arg, 10, 64)
			if err != nil {
				return fmt.Errorf("incorrect id=%s: %w", arg, err)
			}
		}

		if _, err := client.Stop(ctx.Context, &api.IDs{Ids: ids}); err != nil {
			return fmt.Errorf("client.Stop failed: %w", err)
		}

		for _, id := range []uint64{} {
			fmt.Println(id)
		}
		return nil
	},
}
