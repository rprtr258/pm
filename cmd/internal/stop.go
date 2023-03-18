package internal

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/xerr"
)

func init() {
	AllCmds = append(
		AllCmds,
		&cli.Command{
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
		},
	)
}

type stopCmd struct {
	names []string
	tags  []string
	ids   []uint64
}

func (cmd *stopCmd) Validate(ctx *cli.Context, configs []RunConfig) error {
	return nil
}

func (cmd *stopCmd) Run(
	ctx *cli.Context,
	configs []RunConfig,
	client client.Client,
	_ map[db.ProcID]db.ProcData,
	configList map[db.ProcID]db.ProcData,
) error {
	ids := internal.FilterProcs[uint64](
		configList,
		internal.WithGeneric(ctx.Args().Slice()),
		internal.WithIDs(cmd.ids),
		internal.WithNames(cmd.names),
		internal.WithTags(cmd.tags),
		internal.WithAllIfNoFilters,
	)

	if len(ids) == 0 {
		fmt.Println("nothing to stop")
		return nil
	}

	if err := client.Stop(ctx.Context, ids); err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	for _, id := range []uint64{} {
		fmt.Println(id)
	}

	return nil
}
