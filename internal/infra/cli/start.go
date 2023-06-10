package cli

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
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
		startCmd := startCmdProps{
			names:    ctx.StringSlice("name"),
			tags:     ctx.StringSlice("tags"),
			statuses: ctx.StringSlice("status"),
			ids:      ctx.Uint64Slice("id"),
			args:     ctx.Args().Slice(),
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
			return startCmd.Run(ctx.Context, client, list)
		}

		configs, errLoadConfigs := loadConfigs(ctx.String("config"))
		if errLoadConfigs != nil {
			return errLoadConfigs
		}

		names := lo.FilterMap(configs, func(cfg RunConfig, _ int) (string, bool) {
			return cfg.Name.Unpack()
		})

		configList := lo.PickBy(list, func(_ db.ProcID, procData db.ProcData) bool {
			return lo.Contains(names, procData.Name)
		})

		return startCmd.Run(ctx.Context, client, configList)
	},
}

type startCmdProps struct {
	names    []string
	tags     []string
	ids      []uint64
	statuses []string
	args     []string
}

func (cmd *startCmdProps) Run(
	ctx context.Context,
	client client.Client,
	procs map[db.ProcID]db.ProcData,
) error {
	procIDs := core.FilterProcs[uint64](
		procs,
		core.WithAllIfNoFilters,
		core.WithGeneric(cmd.args),
		core.WithIDs(cmd.ids),
		core.WithNames(cmd.names),
		core.WithStatuses(cmd.statuses),
		core.WithTags(cmd.tags),
	)

	if len(procIDs) == 0 {
		fmt.Println("nothing to start")
		return nil
	}

	if err := client.Start(ctx, procIDs); err != nil {
		return xerr.NewWM(err, "client.start")
	}

	fmt.Println(lo.ToAnySlice(procIDs)...)

	return nil
}
