package internal

import (
	"context"
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
				delCmd := deleteCmd{
					names: ctx.StringSlice("name"),
					tags:  ctx.StringSlice("tag"),
					ids:   ctx.Uint64Slice("id"),
					args:  ctx.Args().Slice(),
				}

				client, errList := client.NewGrpcClient()
				if errList != nil {
					return xerr.NewWM(errList, "new grpc client")
				}
				defer deferErr(client.Close)()

				list, errList := client.List(ctx.Context)
				if errList != nil {
					return xerr.NewWM(errList, "server.list")
				}

				if ctx.IsSet("config") {
					return delCmd.Run(ctx.Context, client, list)
				}

				configs, errLoadConfigs := loadConfigs(ctx.String("config"))
				if errLoadConfigs != nil {
					return errLoadConfigs
				}

				names := lo.FilterMap(configs, func(cfg RunConfig, _ int) (string, bool) {
					return cfg.Name.Value, cfg.Name.Valid
				})

				configList := lo.PickBy(list, func(_ db.ProcID, procData db.ProcData) bool {
					return lo.Contains(names, procData.Name)
				})

				return delCmd.Run(ctx.Context, client, configList)
			},
		},
	)
}

type deleteCmd struct {
	names []string
	tags  []string
	ids   []uint64
	args  []string
}

func (cmd *deleteCmd) Run(
	ctx context.Context,
	client client.Client,
	procs map[db.ProcID]db.ProcData,
) error {
	procIDs := internal.FilterProcs[uint64](
		procs,
		internal.WithGeneric(cmd.args),
		internal.WithIDs(cmd.ids),
		internal.WithNames(cmd.names),
		internal.WithTags(cmd.tags),
		internal.WithAllIfNoFilters,
	)

	if len(procIDs) == 0 {
		fmt.Println("Nothing to stop, leaving")
		return nil
	}

	fmt.Printf("Stopping: %v\n", procIDs)

	if err := client.Stop(ctx, procIDs); err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	fmt.Printf("Removing: %v\n", procIDs)

	if errDelete := client.Delete(ctx, procIDs); errDelete != nil {
		return xerr.NewWM(errDelete, "client.delete", xerr.Fields{"procIDs": procIDs})
	}

	return nil
}
