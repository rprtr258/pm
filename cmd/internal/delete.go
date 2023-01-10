package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
)

func init() {
	// TODO: inlines
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
		return executeProcCommand(
			ctx,
			&deleteCmd{
				names: ctx.StringSlice("name"),
				tags:  ctx.StringSlice("tag"),
				ids:   ctx.Uint64Slice("id"),
			},
		)
	},
}

type deleteCmd struct {
	names []string
	tags  []string
	ids   []uint64
}

func (cmd *deleteCmd) Validate(configs []RunConfig) error {
	return nil
}

func (cmd *deleteCmd) Run(
	ctx *cli.Context,
	configs []RunConfig,
	client client.Client,
	list db.DB,
	configList db.DB,
) error {
	// TODO: inline
	return delete(
		ctx.Context,
		configList,
		client,
		ctx.Args().Slice(),
		cmd.names,
		cmd.tags,
		cmd.ids,
	)
}

func delete(
	ctx context.Context,
	filteredDB db.DB,
	client client.Client,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	procIDs := internal.FilterProcs[uint64](
		filteredDB,
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
