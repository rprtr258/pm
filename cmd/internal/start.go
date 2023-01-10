package internal

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
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
		defer commonData.Close()

		return start(
			ctx.Context,
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.StringSlice("status"),
			ctx.Uint64Slice("id"),
		)
	},
}

func start(
	ctx context.Context,
	nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
) error {
	genericFilters := commonData.args

	procIDsToStart := internal.FilterProcs[uint64](
		commonData.filteredDB,
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

	if err := commonData.client.Start(ctx, procIDsToStart); err != nil {
		return fmt.Errorf("client.Start failed: %w", err)
	}

	fmt.Println(lo.ToAnySlice(procIDsToStart)...)

	return nil
}
