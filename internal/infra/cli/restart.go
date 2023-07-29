package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/pkg/client"
)

var _restartCmd = &cli.Command{
	Name:      "restart",
	ArgsUsage: "<name|tag|id|status>...",
	Usage:     "restart already added process(es)",
	Category:  "management",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: "name(s) of process(es) to restart",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "tag(s) of process(es) to restart",
		},
		&cli.Uint64SliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to restart",
		},
		configFlag,
	},
	Action: func(ctx *cli.Context) error {
		if errDaemon := daemon.EnsureRunning(ctx.Context); errDaemon != nil {
			return xerr.NewWM(errDaemon, "ensure daemon is running")
		}

		names := ctx.StringSlice("name")
		tags := ctx.StringSlice("tag")
		ids := ctx.Uint64Slice("id")
		args := ctx.Args().Slice()

		client, errList := client.NewGrpcClient()
		if errList != nil {
			return xerr.NewWM(errList, "new grpc client")
		}
		defer deferErr(client.Close)()

		app, errNewApp := pm.New(client)
		if errNewApp != nil {
			return xerr.NewWM(errNewApp, "new app")
		}

		list, errList := client.List(ctx.Context)
		if errList != nil {
			return xerr.NewWM(errList, "server.list")
		}

		if !ctx.IsSet("config") {
			procIDs := core.FilterProcMap[core.ProcID](
				list,
				core.NewFilter(
					core.WithGeneric(args),
					core.WithIDs(ids),
					core.WithNames(names),
					core.WithTags(tags),
				),
			)

			if len(procIDs) == 0 {
				fmt.Println("nothing to restart")
				return nil
			}

			stoppedProcIDs, err := app.Stop(ctx.Context, procIDs...)
			printIDs("stopped", stoppedProcIDs...)
			if err != nil {
				return xerr.NewWM(err, "client.stop")
			}

			printIDs("starting", procIDs...)
			if errRun := app.Start(ctx.Context, procIDs...); errRun != nil {
				return xerr.NewWM(err, "client.start")
			}

			return nil
		}

		configFile := ctx.String("config")

		configs, errLoadConfigs := core.LoadConfigs(configFile)
		if errLoadConfigs != nil {
			return xerr.NewWM(errLoadConfigs, "load configs", xerr.Fields{
				"config": configFile,
			})
		}

		filteredList, err := app.ListByRunConfigs(ctx.Context, configs)
		if err != nil {
			return xerr.NewWM(err, "list procs by configs")
		}

		// TODO: reuse filter options
		procIDs := core.FilterProcMap[core.ProcID](
			filteredList,
			core.NewFilter(
				core.WithAllIfNoFilters,
				core.WithGeneric(args),
				core.WithIDs(ids),
				core.WithNames(names),
				core.WithTags(tags),
			),
		)

		if len(procIDs) == 0 {
			fmt.Println("nothing to start")
			return nil
		}

		stoppedProcIDs, err := app.Stop(ctx.Context, procIDs...)
		printIDs("stopped", stoppedProcIDs...)
		if err != nil {
			return xerr.NewWM(err, "client.stop")
		}

		printIDs("starting", procIDs...)
		if errRun := app.Start(ctx.Context, procIDs...); errRun != nil {
			return xerr.NewWM(err, "client.start")
		}

		return nil
	},
}
