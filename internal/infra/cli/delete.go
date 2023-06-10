package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/pkg/client"
)

var _deleteCmd = &cli.Command{
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
		names := ctx.StringSlice("name")
		tags := ctx.StringSlice("tag")
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
			procIDs, err := app.Stop(ctx.Context, list, args, names, tags, ids)
			if err != nil {
				return xerr.NewWM(err, "delete")
			}
			fmt.Println("stopped", procIDs)

			if len(procIDs) == 0 {
				fmt.Println("Nothing to stop, leaving")
				return nil
			}

			procIDs, err = app.Delete(ctx.Context, procIDs...)
			if err != nil {
				return xerr.NewWM(err, "delete")
			}
			fmt.Println("deleted", procIDs)

			return nil
		}

		configs, errLoadConfigs := core.LoadConfigs(ctx.String("config"))
		if errLoadConfigs != nil {
			return errLoadConfigs
		}

		list, errList = app.ListByRunConfigs(ctx.Context, configs)
		if errList != nil {
			return xerr.NewWM(errList, "list by run configs", xerr.Fields{"configs": configs})
		}

		procIDs, err := app.Stop(ctx.Context, list, args, names, tags, ids)
		if err != nil {
			return xerr.NewWM(err, "stop")
		}
		fmt.Println("stoped", procIDs)

		procIDs, err = app.Delete(ctx.Context, procIDs...)
		if err != nil {
			return xerr.NewWM(err, "delete")
		}
		fmt.Println("deleted", procIDs)

		return nil
	},
}
