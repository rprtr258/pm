package cli

import (
	"context"
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/internal/infra/daemon"
	"github.com/rprtr258/pm/pkg/client"
)

var _stopCmd = &cli.Command{
	Name:      "stop",
	Usage:     "stop process(es)",
	ArgsUsage: "(id|name|tag|all)...",
	Category:  "management",
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
		&cli.StringSliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to stop",
		},
		configFlag,
	},
	Action: func(ctx *cli.Context) error {
		if errDaemon := daemon.EnsureRunning(ctx.Context); errDaemon != nil {
			return xerr.NewWM(errDaemon, "ensure daemon is running")
		}

		stopCmd := stopCmd{
			names: ctx.StringSlice("name"),
			tags:  ctx.StringSlice("tag"),
			ids: fun.Map[core.PMID](ctx.StringSlice("id"), func(id string) core.PMID {
				return core.PMID(id)
			}),
			args: ctx.Args().Slice(),
		}

		client, errList := client.New()
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
			return xerr.NewWM(errLoadConfigs, "load configs")
		}

		names := fun.FilterMap[string](
			configs,
			func(cfg core.RunConfig) fun.Option[string] {
				return cfg.Name
			})

		configList := lo.PickBy(list, func(_ core.PMID, procData core.Proc) bool {
			return fun.Contains(names, procData.Name)
		})

		return stopCmd.Run(ctx.Context, client, configList)
	},
}

type stopCmd struct {
	names []string
	tags  []string
	ids   []core.PMID
	args  []string
}

func (cmd *stopCmd) Run(
	ctx context.Context,
	pmClient client.Client,
	configList core.Procs,
) error {
	app, errNewApp := pm.New(pmClient)
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	procIDs := core.FilterProcMap(
		configList,
		core.NewFilter(
			core.WithGeneric(cmd.args),
			core.WithIDs(cmd.ids...),
			core.WithNames(cmd.names),
			core.WithTags(cmd.tags),
			core.WithAllIfNoFilters,
		),
	)

	if len(procIDs) == 0 {
		fmt.Println("nothing to stop")
		return nil
	}

	err := app.Stop(ctx, procIDs...)
	if err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	return nil
}
