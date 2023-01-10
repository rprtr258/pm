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
		if ctx.IsSet("config") && isConfigFile(ctx.String("config")) {
			return startConfig(
				ctx.Context,
				configs,
				ctx.Args().Slice(),
				ctx.StringSlice("name"),
				ctx.StringSlice("tags"),
				ctx.StringSlice("status"),
				ctx.Uint64Slice("id"),
			)
		}

		return defaultStart(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.StringSlice("status"),
			ctx.Uint64Slice("id"),
		)
	},
}

func startConfig(
	ctx context.Context,
	configs []RunConfig,
	genericFilters, nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
) error {
	names := make([]string, len(configs))
	for _, config := range configs {
		if !config.Name.Valid {
			continue
		}
		names = append(names, config.Name.Value)
	}

	client, err := client.NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(client.Close)

	list, err := client.List(ctx)
	if err != nil {
		return err
	}

	configList := lo.PickBy(
		list,
		func(_ db.ProcID, procData db.ProcData) bool {
			return lo.Contains(names, procData.Name)
		},
	)

	return start(
		ctx,
		configList, client,
		genericFilters, nameFilters, tagFilters, statusFilters,
		idFilters,
	)
}

func defaultStart(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
) error {
	client, err := client.NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(client.Close)

	resp, err := client.List(ctx)
	if err != nil {
		return err
	}

	return start(
		ctx,
		resp, client,
		genericFilters, nameFilters, tagFilters, statusFilters,
		idFilters,
	)
}

func start(
	ctx context.Context,
	resp db.DB, client client.Client,
	genericFilters, nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
) error {
	procIDsToStart := internal.FilterProcs[uint64](
		resp,
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
