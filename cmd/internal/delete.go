package internal

import (
	"context"
	"fmt"

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
	},
	Action: func(ctx *cli.Context) error {
		return delete(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.Uint64Slice("id"),
		)
	},
}

func delete(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	client, err := NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(client.Close)

	resp, err := client.List(ctx)
	if err != nil {
		return err
	}

	procIDs := internal.FilterProcs[uint64](
		resp,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithTags(tagFilters),
	)

	if len(procIDs) == 0 {
		fmt.Println("Nothing to stop, leaving")
		return nil
	}

	fmt.Printf("Stopping and removing: %v\n", procIDs)

	if err := client.Stop(ctx, procIDs); err != nil {
		return fmt.Errorf("client.Stop failed: %w", err)
	}

	return client.Delete(ctx, procIDs)
}
