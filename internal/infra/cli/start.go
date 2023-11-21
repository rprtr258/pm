package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/daemon"
)

var _cmdStart = &cli.Command{
	Name:      "start",
	ArgsUsage: "<name|tag|id|status>...",
	Usage:     "start already added process(es)",
	Category:  "management",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: "name(s) of process(es) to start",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "tag(s) of process(es) to start",
		},
		&cli.StringSliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to start",
		},
		configFlag,
	},
	Action: func(ctx *cli.Context) error {
		names := ctx.StringSlice("name")
		tags := ctx.StringSlice("tag")
		ids := ctx.StringSlice("id")
		args := ctx.Args().Slice()

		app, errNewApp := daemon.New()
		if errNewApp != nil {
			return xerr.NewWM(errNewApp, "new app")
		}

		list := app.List()

		if !ctx.IsSet("config") {
			procIDs := core.FilterProcMap(
				list,
				core.NewFilter(
					core.WithGeneric(args),
					core.WithIDs(ids...),
					core.WithNames(names),
					core.WithTags(tags),
				),
			)

			if len(procIDs) == 0 {
				fmt.Println("nothing to start")
				return nil
			}

			if err := app.Start(procIDs...); err != nil {
				return xerr.NewWM(err, "client.start")
			}

			printIDs(procIDs...)

			return nil
		}

		configFile := ctx.String("config")

		configs, errLoadConfigs := core.LoadConfigs(configFile)
		if errLoadConfigs != nil {
			return xerr.NewWM(errLoadConfigs, "load configs", xerr.Fields{
				"config": configFile,
			})
		}

		filteredList, err := app.ListByRunConfigs(configs)
		if err != nil {
			return xerr.NewWM(err, "list procs by configs")
		}

		// TODO: reuse filter options
		procIDs := core.FilterProcMap(
			filteredList,
			core.NewFilter(
				core.WithGeneric(args),
				core.WithIDs(ids...),
				core.WithNames(names),
				core.WithTags(tags),
				core.WithAllIfNoFilters,
			),
		)

		if len(procIDs) == 0 {
			fmt.Println("nothing to start")
			return nil
		}

		if err := app.Start(procIDs...); err != nil {
			return xerr.NewWM(err, "client.start")
		}

		printIDs(procIDs...)

		return nil
	},
}
