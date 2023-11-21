package cli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rprtr258/scuf"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	fmt2 "github.com/wissance/stringFormatter"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/daemon"
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
											Case(core.LogTypeStdout, scuf.FgHiWhite).
											Case(core.LogTypeStderr, scuf.FgHiBlack).
											Default(scuf.FgRed)

			fmt.Println(fmt2.FormatComplex(
				// "{at} {proc} {sep} {line}", // TODO: don't show 'at' by default, enable on flag
				"{proc} {sep} {line}",
				map[string]any{
					"at": scuf.String(line.At.In(time.Local).Format("2006-01-02 15:04:05"), scuf.FgHiBlack),
					// TODO: different colors for different IDs
					"proc": scuf.String(line.ProcName, scuf.FgRed),
					"sep":  scuf.String("|", scuf.FgGreen),
					"line": scuf.String(line.Line, lineColor),
				},
			))
		}
	}
}

var _cmdLogs = &cli.Command{
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
		&cli.StringSliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to run",
		},
		configFlag,
	},
	Action: func(ctx *cli.Context) error {
		app, errNewApp := daemon.New()
		if errNewApp != nil {
			return xerr.NewWM(errNewApp, "new app")
		}

		names := ctx.StringSlice("name")
		tags := ctx.StringSlice("tag")
		ids := ctx.StringSlice("id")
		args := ctx.Args().Slice()

		// TODO: filter on server
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
