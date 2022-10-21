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
	ArgsUsage: "<id|name|namespace|all|json|stdin...>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "watch",
			Usage: "Stop watching folder for changes",
		},
		// &cli.BoolFlag{Name:        "shutdown-with-message", Usage: "shutdown an application with process.send('shutdown') instead of process.kill(pid, SIGINT)"},
		// &cli.DurationFlag{Name:    "kill-timeout", Aliases: []string{"k"}, Usage: "delay before sending final SIGKILL signal to process"},
		// &cli.BoolFlag{Name:        "no-treekill", Usage: "Only kill the main process, not detached children"},
	},
	Action: func(ctx *cli.Context) error {
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferFunc()

		// watch := ctx.Bool("watch")

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

		fmt.Println(resp.GetId())
		return nil
	},
}
