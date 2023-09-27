package cli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	fmt2 "github.com/wissance/stringFormatter"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/internal/infra/cli/log/buffer"
	"github.com/rprtr258/pm/internal/infra/daemon"
	"github.com/rprtr258/pm/pkg/client"
)

func mergeChans(ctx context.Context, chans ...<-chan core.LogLine) <-chan core.LogLine {
	out := make(chan core.LogLine)
	go func() {
		var wg sync.WaitGroup
		wg.Add(len(chans))
		for _, ch := range chans {
			go func(ch <-chan core.LogLine) {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case v, ok := <-ch:
						if !ok {
							return
						}

						select {
						case <-ctx.Done():
							return
						case out <- v:
						}
					}
				}
			}(ch)
		}
		wg.Wait()
		close(out)
	}()
	return out
}

func watchLogs(ctx context.Context, ch <-chan core.LogLine) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case line, ok := <-ch:
			if !ok {
				return nil
			}

			if line.Err != nil {
				line.Line = line.Err.Error()
			}

			lineColor := lo.Switch[core.LogType, []byte](line.Type). // all interesting cases are handled
											Case(core.LogTypeStdout, buffer.FgHiWhite).
											Case(core.LogTypeStderr, buffer.FgHiBlack).
											Default(buffer.FgRed)

			fmt.Println(fmt2.FormatComplex(
				// "{at} {proc} {sep} {line}", // TODO: don't show 'at' by default, enable on flag
				"{proc} {sep} {line}",
				map[string]any{
					"at": buffer.String(line.At.In(time.Local).Format("2006-01-02 15:04:05"), buffer.FgHiBlack),
					// TODO: different colors for different IDs
					"proc": buffer.String(fmt.Sprintf("%d|%s", line.ID, line.Name), buffer.FgRed),
					"sep":  buffer.String("|", buffer.FgGreen),
					"line": buffer.String(line.Line, lineColor),
				},
			))
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

		client, errClient := client.New()
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
				fmt.Println("nothing to watch")
				return nil
			}

			logsChs := make([]<-chan core.LogLine, len(procIDs))
			for i, procID := range procIDs {
				logsCh, errLogs := app.Logs(ctx.Context, procID)
				if errLogs != nil {
					return xerr.NewWM(errLogs, "watch procs", xerr.Fields{"procIDs": procIDs})
				}

				logsChs[i] = logsCh
			}

			mergedLogsCh := mergeChans(ctx.Context, logsChs...)

			return watchLogs(ctx.Context, mergedLogsCh)
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
			fmt.Println("nothing to watch")
			return nil
		}

		logsChs := make([]<-chan core.LogLine, len(procIDs))
		for i, procID := range procIDs {
			logsCh, errLogs := app.Logs(ctx.Context, procID)
			if errLogs != nil {
				return xerr.NewWM(errLogs, "watch procs from config", xerr.Fields{"procIDs": procIDs})
			}

			logsChs[i] = logsCh
		}

		mergedLogsCh := mergeChans(ctx.Context, logsChs...)

		return watchLogs(ctx.Context, mergedLogsCh)
	},
}
