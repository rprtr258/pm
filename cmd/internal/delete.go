package internal

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
)

func init() {
	AllCmds = append(
		AllCmds,
		&cli.Command{
			Name:      "delete",
			Aliases:   []string{"del", "rm"},
			Usage:     "stop and remove process(es)",
			ArgsUsage: "<name|id|namespace|tag|json>...",
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:  "name",
					Usage: "name(s) of process(es) to stop and remove",
				},
				&cli.StringSliceFlag{
					Name:  "tag",
					Usage: "tag(s) of process(es) to stop and remove",
				},
				&cli.Uint64SliceFlag{
					Name:  "id",
					Usage: "id(s) of process(es) to stop and remove",
				},
				configFlag,
			},
			Action: func(ctx *cli.Context) error {
				return executeProcCommand(
					ctx,
					&deleteCmd{
						names: ctx.StringSlice("name"),
						tags:  ctx.StringSlice("tag"),
						ids:   ctx.Uint64Slice("id"),
					},
				)
			},
		},
	)
}

type deleteCmd struct {
	names []string
	tags  []string
	ids   []uint64
}

func (cmd *deleteCmd) Validate(ctx *cli.Context, configs []RunConfig) error {
	return nil
}

func (cmd *deleteCmd) Run(
	ctx *cli.Context,
	configs []RunConfig,
	client client.Client,
	_ map[db.ProcID]db.ProcData, // TODO: ???
	procs map[db.ProcID]db.ProcData,
) error {
	procIDs := internal.FilterProcs[uint64](
		procs,
		internal.WithGeneric(ctx.Args().Slice()),
		internal.WithIDs(cmd.ids),
		internal.WithNames(cmd.names),
		internal.WithTags(cmd.tags),
		internal.WithAllIfNoFilters,
	)

	if len(procIDs) == 0 {
		fmt.Println("Nothing to stop, leaving")
		return nil
	}

	fmt.Printf("Stopping: %v\n", procIDs)

	if err := client.Stop(ctx.Context, procIDs); err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	fmt.Printf("Removing: %v\n", procIDs)

	if errDelete := client.Delete(ctx.Context, procIDs); errDelete != nil {
		return xerr.NewWM(errDelete, "client.delete", xerr.Field("procIDs", procIDs))
	}

	return nil
}
