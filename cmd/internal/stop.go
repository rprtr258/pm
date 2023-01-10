package internal

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
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
		configFlag,
	},
	Action: func(ctx *cli.Context) error {
		return executeProcCommand(
			ctx,
			&stopCmd{
				names: ctx.StringSlice("name"),
				tags:  ctx.StringSlice("tag"),
				ids:   ctx.Uint64Slice("id"),
			},
		)
	},
}

type stopCmd struct {
	names []string
	tags  []string
	ids   []uint64
}

func (cmd *stopCmd) Validate(configs []RunConfig) error {
	return nil
}

func (cmd *stopCmd) Run(
	ctx *cli.Context,
	configs []RunConfig,
	client client.Client,
	list db.DB,
	configList db.DB,
) error {
	// TODO: inline
	return stop(
		ctx.Context,
		configList,
		client,
		ctx.Args().Slice(),
		cmd.names,
		cmd.tags,
		cmd.ids,
	)
}

func stop(
	ctx context.Context,
	list db.DB,
	client client.Client,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	ids := internal.FilterProcs[uint64](
		list,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithTags(tagFilters),
		internal.WithAllIfNoFilters,
	)

	if len(ids) == 0 {
		fmt.Println("nothing to stop")
		return nil
	}

	if err := client.Stop(ctx, ids); err != nil {
		return fmt.Errorf("client.Stop failed: %w", err)
	}

	for _, id := range []uint64{} {
		fmt.Println(id)
	}
	return nil
}
