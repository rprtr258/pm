package internal

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
)

func init() {
	AllCmds = append(AllCmds, StopCmd)
}

var StopCmd = &cli.Command{
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
		&cli.StringFlag{
			Name:      "config",
			Usage:     "config file to use",
			Aliases:   []string{"f"},
			TakesFile: true,
		},
	},
	Action: func(ctx *cli.Context) error {
		args := ctx.Args().Slice()

		if ctx.IsSet("config") && isConfigFile(ctx.String("config")) {
			return stopConfig(
				ctx.Context,
				ctx.String("config"),
				args,
				ctx.StringSlice("name"),
				ctx.StringSlice("tags"),
				ctx.Uint64Slice("id"),
			)
		}

		return defaultStop(
			ctx.Context,
			args,
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.Uint64Slice("id"),
		)
	},
}

func stopConfig(
	ctx context.Context,
	configFilename string,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	configs, err := loadConfig(configFilename)
	if err != nil {
		return err
	}

	names := make([]string, len(configs))
	for _, config := range configs {
		if !config.Name.Valid {
			continue
		}
		names = append(names, config.Name.Value)
	}

	client, err := client.NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(client.Close)

	list, err := client.List(ctx)
	if err != nil {
		return err
	}

	configList := lo.PickBy(
		list,
		func(_ db.ProcID, procData db.ProcData) bool {
			return lo.Contains(names, procData.Name)
		},
	)

	return stop(
		ctx,
		configList,
		client,
		genericFilters, nameFilters, tagFilters,
		idFilters,
	)
}

func defaultStop(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	client, err := client.NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(client.Close)

	resp, err := db.New(internal.FileDaemonDB).List()
	if err != nil {
		return err
	}

	return stop(
		ctx,
		resp,
		client,
		genericFilters, nameFilters, tagFilters,
		idFilters,
	)
}

func stop(
	ctx context.Context,
	resp db.DB,
	client client.Client,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	ids := internal.FilterProcs[uint64](
		resp,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithTags(tagFilters),
		internal.WithAllIfNoFilters,
	)

	if len(ids) == 0 {
		fmt.Println("nothing to stop")
		return nil
	}

	if err := client.Stop(ctx, ids); err != nil {
		return fmt.Errorf("client.Stop failed: %w", err)
	}

	for _, id := range []uint64{} {
		fmt.Println(id)
	}
	return nil
}
