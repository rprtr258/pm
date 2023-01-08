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
	},
	Action: func(ctx *cli.Context) error {
		return start(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.StringSlice("status"),
			ctx.Uint64Slice("id"),
		)
	},
}

func start(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
) error {
	resp, err := db.New(internal.FileDaemonDB).List()
	if err != nil {
		return err
	}

	procIDsToStart := internal.FilterProcs[uint64](
		resp,
		internal.WithAllIfNoFilters,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithStatuses(statusFilters),
		internal.WithTags(tagFilters),
	)

	client, err := client.NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(client.Close)

	if err := client.Start(ctx, procIDsToStart); err != nil {
		return fmt.Errorf("client.Start failed: %w", err)
	}

	fmt.Println(lo.ToAnySlice(procIDsToStart)...)

	return nil
}
