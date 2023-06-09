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
			Name:      "stop",
			Usage:     "stop a process",
			ArgsUsage: "<id|name|namespace|all|json>...",
			Flags: []cli.Flag{
				// &cli.BoolFlag{
				// 	Name:  "watch",
				// 	Usage: "stop watching for file changes",
				// },
				// &cli.BoolFlag{
				// 	Name:  "kill",
				// 	Usage: "kill process with SIGKILL instead of SIGINT",
				// },
				// &cli.DurationFlag{
				// 	Name:    "kill-timeout",
				// 	Aliases: []string{"k"},
				// 	Usage:   "delay before sending final SIGKILL signal to process",
				// },
				// &cli.BoolFlag{
				// 	Name:  "no-treekill",
				// 	Usage: "Only kill the main process, not detached children",
				// },
				// TODO: -i/... to confirm which procs will be stopped
				&cli.StringSliceFlag{
					Name:  "name",
					Usage: "name(s) of process(es) to stop",
				},
				&cli.StringSliceFlag{
					Name:  "tag",
					Usage: "tag(s) of process(es) to stop",
				},
				&cli.Uint64SliceFlag{
					Name:  "id",
					Usage: "id(s) of process(es) to stop",
				},
				configFlag,
			},
			Action: func(ctx *cli.Context) error {
				stopCmd := stopCmd{
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
					return executeProcCommandWithConfig5(ctx, client, stopCmd, ctx.String("config"))
				}

				return executeProcCommandWithoutConfig5(ctx, client, stopCmd)
			},
		},
	)
}

type stopCmd struct {
	names []string
	tags  []string
	ids   []uint64
}

func (cmd *stopCmd) Validate(ctx *cli.Context, configs []RunConfig) error {
	return nil
}

func (cmd *stopCmd) Run(
	ctx *cli.Context,
	client client.Client,
	configList map[db.ProcID]db.ProcData,
) error {
	ids := internal.FilterProcs[uint64](
		configList,
		internal.WithGeneric(ctx.Args().Slice()),
		internal.WithIDs(cmd.ids),
		internal.WithNames(cmd.names),
		internal.WithTags(cmd.tags),
		internal.WithAllIfNoFilters,
	)

	if len(ids) == 0 {
		fmt.Println("nothing to stop")
		return nil
	}

	if err := client.Stop(ctx.Context, ids); err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	for _, id := range []uint64{} {
		fmt.Println(id)
	}

	return nil
}

func executeProcCommandWithoutConfig5(ctx *cli.Context, client client.Client, cmd stopCmd) error {
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

func executeProcCommandWithConfig5(
	ctx *cli.Context,
	client client.Client,
	cmd stopCmd,
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
