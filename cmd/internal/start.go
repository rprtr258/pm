package internal

import (
	"fmt"

	"github.com/samber/lo"
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
			Name:      "start",
			ArgsUsage: "<name|tag|id|status>...",
			Usage:     "start process and manage it",
			Flags: []cli.Flag{
				// &cli.BoolFlag{Name:        "only", Usage: "with json declaration, allow to only act on one application"},
				&cli.StringSliceFlag{
					Name:  "name",
					Usage: "name(s) of process(es) to run",
				},
				&cli.StringSliceFlag{
					Name:  "tag",
					Usage: "tag(s) of process(es) to run",
				},
				&cli.Uint64SliceFlag{
					Name:  "id",
					Usage: "id(s) of process(es) to run",
				},
				&cli.StringSliceFlag{
					Name:  "status",
					Usage: "status(es) of process(es) to run",
				},
				configFlag,
			},
			Action: func(ctx *cli.Context) error {
				return executeProcCommand(
					ctx,
					&startCmd{
						names:    ctx.StringSlice("name"),
						tags:     ctx.StringSlice("tags"),
						statuses: ctx.StringSlice("status"),
						ids:      ctx.Uint64Slice("id"),
					},
				)
			},
		},
	)
}

type startCmd struct {
	names    []string
	tags     []string
	ids      []uint64
	statuses []string
}

func (cmd *startCmd) Validate(ctx *cli.Context, configs []RunConfig) error {
	return nil
}

func (cmd *startCmd) Run(
	ctx *cli.Context,
	configs []RunConfig,
	client client.Client,
	_ map[db.ProcID]db.ProcData,
	procs map[db.ProcID]db.ProcData,
) error {
	procIDsToStart := internal.FilterProcs[uint64](
		procs,
		internal.WithAllIfNoFilters,
		internal.WithGeneric(ctx.Args().Slice()),
		internal.WithIDs(cmd.ids),
		internal.WithNames(cmd.names),
		internal.WithStatuses(cmd.statuses),
		internal.WithTags(cmd.tags),
	)

	if len(procIDsToStart) == 0 {
		fmt.Println("nothing to start")
		return nil
	}

	if err := client.Start(ctx.Context, procIDsToStart); err != nil {
		return xerr.NewWM(err, "client.start")
	}

	fmt.Println(lo.ToAnySlice(procIDsToStart)...)

	return nil
}
