package internal

import (
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
				}

				client, errList := client.NewGrpcClient()
				if errList != nil {
					return xerr.NewWM(errList, "new grpc client")
				}
				defer deferErr(client.Close)()

				if ctx.IsSet("config") {
					return executeProcCommandWithConfig3(ctx, client, delCmd, ctx.String("config"))
				}

				return executeProcCommandWithoutConfig3(ctx, client, delCmd)
			},
		},
	)
}

type deleteCmd struct {
	names []string
	tags  []string
	ids   []uint64
}

func (cmd *deleteCmd) Validate(ctx *cli.Context, configs []RunConfig) error {
	return nil
}

func (cmd *deleteCmd) Run(
	ctx *cli.Context,
	client client.Client,
	procs map[db.ProcID]db.ProcData,
) error {
	procIDs := internal.FilterProcs[uint64](
		procs,
		internal.WithGeneric(ctx.Args().Slice()),
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

	if err := client.Stop(ctx.Context, procIDs); err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	fmt.Printf("Removing: %v\n", procIDs)

	if errDelete := client.Delete(ctx.Context, procIDs); errDelete != nil {
		return xerr.NewWM(errDelete, "client.delete", xerr.Fields{"procIDs": procIDs})
	}

	return nil
}

func executeProcCommandWithoutConfig3(ctx *cli.Context, client client.Client, cmd deleteCmd) error {
	list, errList := client.List(ctx.Context)
	if errList != nil {
		return xerr.NewWM(errList, "server.list")
	}

	if errRun := cmd.Run(
		ctx,
		client,
		list,
	); errRun != nil {
		return xerr.NewWM(errRun, "run")
	}

	return nil
}

func executeProcCommandWithConfig3(
	ctx *cli.Context,
	client client.Client,
	cmd deleteCmd,
	configFilename string,
) error {
	list, errList := client.List(ctx.Context)
	if errList != nil {
		return xerr.NewWM(errList, "server.list")
	}

	configs, errLoadConfigs := loadConfigs(configFilename)
	if errLoadConfigs != nil {
		return errLoadConfigs
	}

	if err := cmd.Validate(ctx, configs); err != nil {
		return xerr.NewWM(err, "validate config")
	}

	names := lo.FilterMap(configs, func(cfg RunConfig, _ int) (string, bool) {
		return cfg.Name.Value, cfg.Name.Valid
	})

	configList := lo.PickBy(list, func(_ db.ProcID, procData db.ProcData) bool {
		return lo.Contains(names, procData.Name)
	})

	if errRun := cmd.Run(
		ctx,
		client,
		configList,
	); errRun != nil {
		return xerr.NewWM(errRun, "run config list")
	}

	return nil
}
