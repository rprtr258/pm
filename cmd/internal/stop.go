package internal

import (
	"errors"
	"fmt"

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
	},
	Action: func(ctx *cli.Context) error {
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferFunc()

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("what to stop expected")
		}

		resp, err := client.Stop(ctx.Context, &api.DeleteReq{
			Filters: []*api.DeleteFilter{{
				// Filter: &api.DeleteFilter_Name{Name: name},
				// Filter: &api.DeleteFilter_Tags{Tags: &api.Tags{Tags: []string{}}},
			}},
		})
		if err != nil {
			return err
		}

		for _, id := range resp.GetId() {
			fmt.Println(id)
		}
		return nil
	},
}
