package internal

import (
	"context"
	"fmt"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/db"
	"github.com/urfave/cli/v2"
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
	client, deferFunc, err := NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(deferFunc)

	dbHandle := db.New(_daemonDBFile)

	resp, err := dbHandle.List()
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

	if _, err := client.Stop(ctx, &api.IDs{Ids: procIDs}); err != nil {
		return fmt.Errorf("client.Stop failed: %w", err)
	}

	if err := dbHandle.Delete(procIDs); err != nil {
		return err // TODO: add errs descriptions, loggings
	}

	// TODO: delete log files too
	// 	_, err := os.Stat(string(*f))
	// 	if err != nil {
	// 		return true
	// 	}
	// 	err = os.Remove(string(*f))

	return nil

}
