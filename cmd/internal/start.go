package internal

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
)

func init() {
	AllCmds = append(AllCmds, StartCmd)
}

var StartCmd = &cli.Command{
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
}

var _ = procCommand(&startCmd{})

type startCmd struct {
	names    []string
	tags     []string
	ids      []uint64
	statuses []string
}

func (cmd *startCmd) Validate(configs []RunConfig) error {
	return nil
}

func (cmd *startCmd) Run(
	ctx *cli.Context,
	configs []RunConfig,
	client client.Client,
	list db.DB,
	configList db.DB,
) error {
	// TODO: inline
	return start(
		ctx.Context,
		configList,
		client,
		ctx.Args().Slice(),
		cmd.names,
		cmd.tags,
		cmd.statuses,
		cmd.ids,
	)
}

func start(
	ctx context.Context,
	filteredDB db.DB,
	client client.Client,
	genericFilters, nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
) error {
	procIDsToStart := internal.FilterProcs[uint64](
		filteredDB,
		internal.WithAllIfNoFilters,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithStatuses(statusFilters),
		internal.WithTags(tagFilters),
	)

	if len(procIDsToStart) == 0 {
		fmt.Println("nothing to start")
		return nil
	}

	if err := client.Start(ctx, procIDsToStart); err != nil {
		return fmt.Errorf("client.Start failed: %w", err)
	}

	fmt.Println(lo.ToAnySlice(procIDsToStart)...)

	return nil
}
