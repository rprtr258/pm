package cli

import (
	"fmt"
	"strings"
	"sync"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/spf13/cobra"
	fmt2 "github.com/wissance/stringFormatter"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
)

func getProcs(
	app app.App,
	rest, ids, names, tags []string,
	config *string,
) ([]core.Proc, error) {
	filterFunc := core.FilterFunc(
		core.WithGeneric(rest...),
		core.WithIDs(ids...),
		core.WithNames(names...),
		core.WithTags(tags...),
		core.WithAllIfNoFilters,
	)

	if config == nil {
		return app.
			List().
			Filter(filterFunc).
			ToSlice(), nil
	}

	configs, errLoadConfigs := core.LoadConfigs(string(*config))
	if errLoadConfigs != nil {
		return nil, errors.Wrapf(errLoadConfigs, "load configs: %v", *config)
	}

	return app.
		ListByRunConfigs(configs).
		Filter(filterFunc).
		ToSlice(), nil
}

var _cmdLogs = func() *cobra.Command {
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "logs [name|tag|id]...",
		Short:             "watch for processes logs",
		GroupID:           "inspection",
		ValidArgsFunction: completeArgGenericSelector,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			app, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrapf(errNewApp, "new app")
			}

			procs, err := getProcs(app, args, ids, names, tags, config)
			if err != nil {
				return errors.Wrapf(err, "get proc ids")
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
					return errors.Wrapf(errLogs, "watch procs: %v", proc)
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
						Case(scuf.FgHiWhite, core.LogTypeStdout).
						Case(scuf.FgHiBlack, core.LogTypeStderr).
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
		},
	}
	//   .option('--json', 'json log output')
	//   .option('--format', 'formated log output')
	//   .option('--raw', 'raw output')
	//   .option('--err', 'only shows error output')
	//   .option('--out', 'only shows standard output')
	//   .option('--lines <n>', 'output the last N lines, instead of the last 15 by default')
	//   .option('--timestamp [format]', 'add timestamps (default format YYYY-MM-DD-HH:mm:ss)')
	//   .option('--highlight', 'enable highlighting')
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()

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
