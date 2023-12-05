package cli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rprtr258/scuf"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	fmt2 "github.com/wissance/stringFormatter"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
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

type _cmdLogs struct {
	Names []flagProcName `long:"name" description:"name(s) of process(es) to list"`
	Tags  []flagProcTag  `long:"tag" description:"tag(s) of process(es) to list"`
	IDs   []flagPMID     `long:"id" description:"id(s) of process(es) to list"`
	Args  struct {
		Rest []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
	configFlag
}

func (x *_cmdLogs) Execute(_ []string) error {
	ctx := context.TODO()

	app, errNewApp := app.New()
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	list := app.List()

	if x.configFlag.Config == nil {
		procIDs := core.FilterProcMap(
			list,
			core.WithGeneric(x.Args.Rest...),
			core.WithIDs(x.IDs...),
			core.WithNames(x.Names...),
			core.WithTags(x.Tags...),
		)
		if len(procIDs) == 0 {
			fmt.Println("nothing to watch")
			return nil
		}

		logsChs := make([]<-chan core.LogLine, len(procIDs))
		for i, procID := range procIDs {
			logsCh, errLogs := app.Logs(ctx, procID)
			if errLogs != nil {
				return xerr.NewWM(errLogs, "watch procs", xerr.Fields{"procIDs": procIDs})
			}

			logsChs[i] = logsCh
		}

		mergedLogsCh := mergeChans(ctx, logsChs...)

		return watchLogs(ctx, mergedLogsCh)
	}

	configs, errLoadConfigs := core.LoadConfigs(string(*x.Config))
	if errLoadConfigs != nil {
		return xerr.NewWM(errLoadConfigs, "load configs", xerr.Fields{
			"config": *x.Config,
		})
	}

	filteredList, err := app.ListByRunConfigs(configs)
	if err != nil {
		return xerr.NewWM(err, "list procs by configs")
	}

	// TODO: reuse filter options
	procIDs := core.FilterProcMap(
		filteredList,
		core.WithGeneric(x.Args.Rest...),
		core.WithIDs(x.IDs...),
		core.WithNames(x.Names...),
		core.WithTags(x.Tags...),
		core.WithAllIfNoFilters,
	)

	if len(procIDs) == 0 {
		fmt.Println("nothing to watch")
		return nil
	}

	logsChs := make([]<-chan core.LogLine, len(procIDs))
	for i, procID := range procIDs {
		logsCh, errLogs := app.Logs(ctx, procID)
		if errLogs != nil {
			return xerr.NewWM(errLogs, "watch procs from config", xerr.Fields{"procIDs": procIDs})
		}

		logsChs[i] = logsCh
	}

	mergedLogsCh := mergeChans(ctx, logsChs...)

	return watchLogs(ctx, mergedLogsCh)
}
