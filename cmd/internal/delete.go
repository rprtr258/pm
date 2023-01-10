package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
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
		defer commonData.Close()

		return delete(
			ctx.Context,
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.Uint64Slice("id"),
		)
	},
}

func delete(
	ctx context.Context,
	nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	genericFilters := commonData.args

	procIDs := internal.FilterProcs[uint64](
		commonData.filteredDB,
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

	if err := commonData.client.Stop(ctx, procIDs); err != nil {
		log.Println(fmt.Errorf("client.Stop failed: %w", err).Error())
	}

	return commonData.client.Delete(ctx, procIDs)
}
