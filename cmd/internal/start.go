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
				startCmd := startCmd{
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

				if ctx.IsSet("config") {
					return executeProcCommandWithConfig4(ctx.Context, client, startCmd, ctx.String("config"))
				}

				return executeProcCommandWithoutConfig4(ctx.Context, client, startCmd)
			},
		},
	)
}

type startCmd struct {
	names    []string
	tags     []string
	ids      []uint64
	statuses []string
	args     []string
}

func (cmd *startCmd) Validate(configs []RunConfig) error {
	return nil
}

func (cmd *startCmd) Run(
	ctx context.Context,
	client client.Client,
	procs map[db.ProcID]db.ProcData,
) error {
	procIDsToStart := internal.FilterProcs[uint64](
		procs,
		internal.WithAllIfNoFilters,
		internal.WithGeneric(cmd.args),
		internal.WithIDs(cmd.ids),
		internal.WithNames(cmd.names),
		internal.WithStatuses(cmd.statuses),
		internal.WithTags(cmd.tags),
	)

	if len(procIDsToStart) == 0 {
		fmt.Println("nothing to start")
		return nil
	}

	if err := client.Start(ctx, procIDsToStart); err != nil {
		return xerr.NewWM(err, "client.start")
	}

	fmt.Println(lo.ToAnySlice(procIDsToStart)...)

	return nil
}

func executeProcCommandWithoutConfig4(ctx context.Context, client client.Client, cmd startCmd) error {
	list, errList := client.List(ctx)
	if errList != nil {
		return xerr.NewWM(errList, "server.list")
	}

	if errRun := cmd.Run(ctx, client, list); errRun != nil {
		return xerr.NewWM(errRun, "run")
	}

	return nil
}

func executeProcCommandWithConfig4(
	ctx context.Context,
	client client.Client,
	cmd startCmd,
	configFilename string,
) error {
	list, errList := client.List(ctx)
	if errList != nil {
		return xerr.NewWM(errList, "server.list")
	}

	configs, errLoadConfigs := loadConfigs(configFilename)
	if errLoadConfigs != nil {
		return errLoadConfigs
	}

	if err := cmd.Validate(configs); err != nil {
		return xerr.NewWM(err, "validate config")
	}

	names := lo.FilterMap(configs, func(cfg RunConfig, _ int) (string, bool) {
		return cfg.Name.Value, cfg.Name.Valid
	})

	configList := lo.PickBy(list, func(_ db.ProcID, procData db.ProcData) bool {
		return lo.Contains(names, procData.Name)
	})

	if errRun := cmd.Run(ctx, client, configList); errRun != nil {
		return xerr.NewWM(errRun, "run config list")
	}

	return nil
}
