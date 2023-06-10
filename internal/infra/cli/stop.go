package cli

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/pkg/client"
)

var _stopCmd = &cli.Command{
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

		if !ctx.IsSet("config") {
			return stopCmd.Run(ctx.Context, client, list)
		}

		configs, errLoadConfigs := core.LoadConfigs(ctx.String("config"))
		if errLoadConfigs != nil {
			return errLoadConfigs
		}

		names := lo.FilterMap(configs, func(cfg core.RunConfig, _ int) (string, bool) {
			return cfg.Name.Unpack()
		})

		configList := lo.PickBy(list, func(_ core.ProcID, procData core.ProcData) bool {
			return lo.Contains(names, procData.Name)
		})

		return stopCmd.Run(ctx.Context, client, configList)
	},
}

type stopCmd struct {
	names []string
	tags  []string
	ids   []uint64
	args  []string
}

func (cmd *stopCmd) Run(
	ctx context.Context,
	client client.Client,
	configList map[core.ProcID]core.ProcData,
) error {
	app := pm.New(client)

	ids := core.FilterProcs[core.ProcID](
		configList,
		core.WithGeneric(cmd.args),
		core.WithIDs(cmd.ids),
		core.WithNames(cmd.names),
		core.WithTags(cmd.tags),
		core.WithAllIfNoFilters,
	)

	if len(ids) == 0 {
		fmt.Println("nothing to stop")
		return nil
	}

	procIDs, err := app.Stop(ctx, configList, cmd.args, cmd.names, cmd.tags, cmd.ids)
	if err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	fmt.Println(lo.ToAnySlice(procIDs)...)

	return nil
}
