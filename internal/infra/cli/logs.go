package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/rprtr258/xerr"
	"github.com/urfave/cli/v2"
	fmt2 "github.com/wissance/stringFormatter"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/pkg/client"
)

func watchLogs(ctx context.Context, ch <-chan core.ProcLogs) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case procLines, ok := <-ch:
			if !ok {
				return nil
			}

			for _, line := range procLines.Lines {
				// TODO: color package does not allow to use format with color parameter,
				// so we carry whole formatting function
				lineType := color.RedString
				switch line.Type { //nolint:exhaustive // all interesting cases are handled
				case core.LogTypeStdout:
					lineType = color.HiWhiteString
				case core.LogTypeStderr:
					lineType = color.HiBlackString
				}

				fmt.Println(fmt2.FormatComplex(
					"{at} {proc} {sep} {line}",
					map[string]any{
						"at": color.HiBlackString("%s", line.At.In(time.Local).Format("2006-01-02 15:04:05")),
						// TODO: different colors for different IDs
						// TODO: pass proc name
						"proc": color.RedString("%d|%s", procLines.ID, "proc-name"),
						"sep":  color.GreenString("%s", "|"),
						"line": lineType(line.Line),
					},
				))
			}
		}
	}
}

var _logsCmd = &cli.Command{
	Name:      "logs",
	ArgsUsage: "<name|tag|id|status>...",
	Usage:     "watch for processes logs",
	Category:  "inspection",
	Flags: []cli.Flag{
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
		configFlag,
	},
	Action: func(ctx *cli.Context) error {
		if errDaemon := daemon.EnsureRunning(ctx.Context); errDaemon != nil {
			return xerr.NewWM(errDaemon, "ensure daemon is running")
		}

		client, errClient := client.NewGrpcClient()
		if errClient != nil {
			return xerr.NewWM(errClient, "new grpc client")
		}
		defer deferErr(client.Close)()

		app, errNewApp := pm.New(client)
		if errNewApp != nil {
			return xerr.NewWM(errNewApp, "new app")
		}

		names := ctx.StringSlice("name")
		tags := ctx.StringSlice("tag")
		ids := ctx.Uint64Slice("id")
		args := ctx.Args().Slice()

		// TODO: filter on server
		list, errList := client.List(ctx.Context)
		if errList != nil {
			return xerr.NewWM(errList, "server.list")
		}

		if !ctx.IsSet("config") {
			procIDs := core.FilterProcMap[core.ProcID](
				list,
				core.NewFilter(
					core.WithGeneric(args),
					core.WithIDs(ids...),
					core.WithNames(names),
					core.WithTags(tags),
				),
			)

			if len(procIDs) == 0 {
				fmt.Println("nothing to watch")
				return nil
			}

			logsCh, errLogs := app.Logs(ctx.Context, procIDs...)
			if errLogs != nil {
				return xerr.NewWM(errLogs, "watch procs", xerr.Fields{"procIDs": procIDs})
			}

			return watchLogs(ctx.Context, logsCh)
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
				core.WithGeneric(args),
				core.WithIDs(ids...),
				core.WithNames(names),
				core.WithTags(tags),
				core.WithAllIfNoFilters,
			),
		)

		if len(procIDs) == 0 {
			fmt.Println("nothing to watch")
			return nil
		}

		logsCh, errLogs := app.Logs(ctx.Context, procIDs...)
		if errLogs != nil {
			return xerr.NewWM(errLogs, "watch procs from config", xerr.Fields{"procIDs": procIDs})
		}

		return watchLogs(ctx.Context, logsCh)
	},
}
