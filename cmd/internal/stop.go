package internal

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/db"
)

func init() {
	AllCmds = append(AllCmds, StopCmd)
}

var StopCmd = &cli.Command{
	Name:      "stop",
	Usage:     "stop a process",
	ArgsUsage: "<id|name|namespace|all|json>...",
	Flags: []cli.Flag{
		// &cli.BoolFlag{
		// 	Name:  "watch",
		// 	Usage: "stop watching for file changes",
		// },
		// &cli.BoolFlag{
		// 	Name:  "kill",
		// 	Usage: "kill process with SIGKILL instead of SIGINT",
		// },
		// &cli.DurationFlag{
		// 	Name:    "kill-timeout",
		// 	Aliases: []string{"k"},
		// 	Usage:   "delay before sending final SIGKILL signal to process",
		// },
		// &cli.BoolFlag{
		// 	Name:  "no-treekill",
		// 	Usage: "Only kill the main process, not detached children",
		// },
		// TODO: -i/... to confirm which procs will be stopped
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: "name(s) of process(es) to stop",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "tag(s) of process(es) to stop",
		},
		&cli.Uint64SliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to stop",
		},
	},
	Action: func(ctx *cli.Context) error {
		return stop(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.Uint64Slice("id"),
		)
	},
}

func stop(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	client, err := NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(client.Close)

	resp, err := db.New(_daemonDBFile).List()
	if err != nil {
		return err
	}

	ids := internal.FilterProcs[uint64](
		resp,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithTags(tagFilters),
	)

	if err := client.Stop(ctx, ids); err != nil {
		return fmt.Errorf("client.Stop failed: %w", err)
	}

	for _, id := range []uint64{} {
		fmt.Println(id)
	}
	return nil
}
