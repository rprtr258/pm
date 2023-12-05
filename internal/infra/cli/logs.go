package cli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/rprtr258/xerr"
	fmt2 "github.com/wissance/stringFormatter"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type _cmdLogs struct {
	Names []flagProcName `long:"name" description:"name(s) of process(es) to list"`
	Tags  []flagProcTag  `long:"tag" description:"tag(s) of process(es) to list"`
	IDs   []flagPMID     `long:"id" description:"id(s) of process(es) to list"`
	Args  struct {
		Rest []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
	configFlag
}

func (x *_cmdLogs) getProcs(app app.App) ([]core.PMID, error) {
	list := app.List()

	if x.configFlag.Config == nil {
		return core.FilterProcMap(
			list,
			core.WithGeneric(x.Args.Rest...),
			core.WithIDs(x.IDs...),
			core.WithNames(x.Names...),
			core.WithTags(x.Tags...),
		), nil
	}

	configs, errLoadConfigs := core.LoadConfigs(string(*x.Config))
	if errLoadConfigs != nil {
		return nil, xerr.NewWM(errLoadConfigs, "load configs", xerr.Fields{
			"config": *x.Config,
		})
	}

	filteredList, err := app.ListByRunConfigs(configs)
	if err != nil {
		return nil, xerr.NewWM(err, "list procs by configs")
	}

	// TODO: reuse filter options
	return core.FilterProcMap(
		filteredList,
		core.WithGeneric(x.Args.Rest...),
		core.WithIDs(x.IDs...),
		core.WithNames(x.Names...),
		core.WithTags(x.Tags...),
		core.WithAllIfNoFilters,
	), nil
}

func (x *_cmdLogs) Execute(_ []string) error {
	ctx := context.TODO()

	app, errNewApp := app.New()
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	procIDs, err := x.getProcs(app)
	if err != nil {
		return xerr.NewWM(err, "get proc ids")
	}
	if len(procIDs) == 0 {
		fmt.Println("nothing to watch")
		return nil
	}

	logsChs := make([]<-chan core.LogLine, len(procIDs))
	for i, procID := range procIDs {
		logsCh, errLogs := app.Logs(ctx, procID)
		if errLogs != nil {
			return xerr.NewWM(errLogs, "watch procs", xerr.Fields{"procID": procID})
		}

		logsChs[i] = logsCh
	}

	mergedLogsCh := make(chan core.LogLine)
	var wg sync.WaitGroup
	for _, ch := range logsChs {
		wg.Add(1)
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
					case mergedLogsCh <- v:
					}
				}
			}
		}(ch)
	}
	go func() {
		wg.Wait()
		close(mergedLogsCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case line, ok := <-mergedLogsCh:
			if !ok {
				return nil
			}

			if line.Err != nil {
				line.Line = line.Err.Error()
			}

			lineColor := fun.Switch(line.Type, scuf.FgRed).
				Case(core.LogTypeStdout, scuf.FgHiWhite).
				Case(core.LogTypeStderr, scuf.FgHiBlack).
				End()

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
