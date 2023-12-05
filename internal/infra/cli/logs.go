package cli

import (
	"context"
	"fmt"
	"strings"
	"sync"

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

func (x *_cmdLogs) getProcs(app app.App) ([]core.Proc, error) {
	filterFunc := core.FilterFunc(
		core.WithGeneric(x.Args.Rest...),
		core.WithIDs(x.IDs...),
		core.WithNames(x.Names...),
		core.WithTags(x.Tags...),
		core.WithAllIfNoFilters,
	)

	if x.configFlag.Config == nil {
		return app.
			List().
			Filter(filterFunc).
			ToSlice(), nil
	}

	configs, errLoadConfigs := core.LoadConfigs(string(*x.Config))
	if errLoadConfigs != nil {
		return nil, xerr.NewWM(errLoadConfigs, "load configs", xerr.Fields{
			"config": *x.Config,
		})
	}

	return app.
		ListByRunConfigs(configs).
		Filter(filterFunc).
		ToSlice(), nil
}

var colors = [...]scuf.Modifier{
	scuf.FgHiRed,
	scuf.FgHiGreen,
	scuf.FgHiYellow,
	scuf.FgHiBlue,
	scuf.FgHiMagenta,
	scuf.FgHiCyan,
	scuf.FgHiWhite,
}

func colorByID(id core.PMID) scuf.Modifier {
	x := 0
	for i := 0; i < len(id); i++ {
		x += int(id[i])
	}
	return colors[x%len(colors)]
}

func (x *_cmdLogs) Execute(_ []string) error {
	ctx := context.TODO()

	app, errNewApp := app.New()
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	procs, err := x.getProcs(app)
	if err != nil {
		return xerr.NewWM(err, "get proc ids")
	}
	if len(procs) == 0 {
		fmt.Println("nothing to watch")
		return nil
	}

	var wg sync.WaitGroup
	mergedLogsCh := make(chan core.LogLine)
	for _, proc := range procs {
		logsCh, errLogs := app.Logs(ctx, proc)
		if errLogs != nil {
			return xerr.NewWM(errLogs, "watch procs", xerr.Fields{"procID": proc})
		}

		wg.Add(1)
		ch := logsCh
		go func() {
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
		}()
	}
	go func() {
		wg.Wait()
		close(mergedLogsCh)
	}()

	pad := 0
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

			pad = max(pad, len(line.ProcName))
			fmt.Println(fmt2.FormatComplex(
				"{proc} {pad}{sep} {line}",
				map[string]any{
					"proc": scuf.String(line.ProcName, colorByID(line.ProcID)),
					"sep":  scuf.String("|", scuf.FgGreen),
					"line": scuf.String(line.Line, lineColor),
					"pad":  strings.Repeat(" ", pad-len(line.ProcName)+1),
				},
			))
		}
	}
}
