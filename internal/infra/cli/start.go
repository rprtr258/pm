package cli

import (
	"fmt"

	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/pkg/client"
)

var _startCmd = &cli.Command{
	Name:      "start",
	ArgsUsage: "<name|tag|id|status>...",
	Usage:     "start process and manage it",
	Flags: []cli.Flag{
		// &cli.BoolFlag{Name:        "only", Usage: "with json declaration, allow to only act on one application"},
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: "name(s) of process(es) to run",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "tag(s) of process(es) to run",
		},
		&cli.Uint64SliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to run",
		},
		&cli.StringSliceFlag{
			Name:  "status",
			Usage: "status(es) of process(es) to run",
		},
		configFlag,
	},
	Action: func(ctx *cli.Context) error {
		names := ctx.StringSlice("name")
		tags := ctx.StringSlice("tags")
		statuses := ctx.StringSlice("status")
		ids := ctx.Uint64Slice("id")
		args := ctx.Args().Slice()

		client, errList := client.NewGrpcClient()
		if errList != nil {
			return xerr.NewWM(errList, "new grpc client")
		}
		defer deferErr(client.Close)()

		app := pm.New(client)

		list, errList := client.List(ctx.Context)
		if errList != nil {
			return xerr.NewWM(errList, "server.list")
		}

		if !ctx.IsSet("config") {
			procIDs := core.FilterProcs[db.ProcID](
				list,
				core.WithAllIfNoFilters,
				core.WithGeneric(args),
				core.WithIDs(ids),
				core.WithNames(names),
				core.WithStatuses(statuses),
				core.WithTags(tags),
			)

			if len(procIDs) == 0 {
				fmt.Println("nothing to start")
				return nil
			}

			if err := app.Start(ctx.Context, procIDs...); err != nil {
				return xerr.NewWM(err, "client.start")
			}
		}

		configs, errLoadConfigs := loadConfigs(ctx.String("config"))
		if errLoadConfigs != nil {
			return errLoadConfigs
		}

		filteredList, err := app.ListByRunConfigs(ctx.Context, configs)
		if err != nil {
			return err
		}

		// TODO: reuse filter options
		procIDs := core.FilterProcs[db.ProcID](
			filteredList,
			core.WithAllIfNoFilters,
			core.WithGeneric(args),
			core.WithIDs(ids),
			core.WithNames(names),
			core.WithStatuses(statuses),
			core.WithTags(tags),
		)

		if len(procIDs) == 0 {
			fmt.Println("nothing to start")
			return nil
		}

		if err := app.Start(ctx.Context, procIDs...); err != nil {
			return xerr.NewWM(err, "client.start")
		}

		fmt.Println(lo.ToAnySlice(procIDs)...)

		return nil
	},
}
