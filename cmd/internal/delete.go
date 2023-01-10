package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
)

func init() {
	AllCmds = append(AllCmds, DeleteCmd)
}

var DeleteCmd = &cli.Command{
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
		if configs != nil {
			return deleteConfig(
				ctx.Context,
				configs,
				ctx.Args().Slice(),
				ctx.StringSlice("name"),
				ctx.StringSlice("tags"),
				ctx.Uint64Slice("id"),
			)
		}

		return defaultDelete(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.Uint64Slice("id"),
		)
	},
}

func deleteConfig(
	ctx context.Context,
	configs []RunConfig,
	genericFilters, nameFilters, tagFilters []string,
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

	return delete(
		ctx,
		configList,
		client,
		genericFilters, nameFilters, tagFilters,
		idFilters,
	)
}

func defaultDelete(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters []string,
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

	return delete(
		ctx,
		resp, client,
		genericFilters, nameFilters, tagFilters,
		idFilters,
	)
}

func delete(
	ctx context.Context,
	resp db.DB, client client.Client,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	procIDs := internal.FilterProcs[uint64](
		resp,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithTags(tagFilters),
		internal.WithAllIfNoFilters,
	)

	if len(procIDs) == 0 {
		fmt.Println("Nothing to stop, leaving")
		return nil
	}

	fmt.Printf("Stopping and removing: %v\n", procIDs)

	if err := client.Stop(ctx, procIDs); err != nil {
		log.Println(fmt.Errorf("client.Stop failed: %w", err).Error())
	}

	return client.Delete(ctx, procIDs)
}
