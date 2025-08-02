package cli

import (
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/errors"
)

var _cmdTUI = func() *cobra.Command {
	const filter = filterAll
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "tui [name|tag|id]...",
		Short:             "open TUI dashboard process(es)",
		ValidArgsFunction: completeArgGenericSelector(filter),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			list := listProcs(dbb)

			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrap(errLoadConfigs, "load configs")
				}

				namesFilter := fun.Map[string](func(cfg core.RunConfig) string {
					return cfg.Name
				}, configs...)

				list = list.
					Filter(func(proc core.ProcStat) bool {
						return fun.Contains(proc.Name, namesFilter...)
					})
			}

			filterFunc := core.FilterFunc(
				core.WithGeneric(args...),
				core.WithIDs(ids...),
				core.WithNames(names...),
				core.WithTags(tags...),
				core.WithAllIfNoFilters,
			)
			procs := list.
				Filter(func(ps core.ProcStat) bool {
					return filterFunc(ps.Proc)
				}).
				Slice()
			if len(procs) == 0 {
				fmt.Println("no procs")
				return nil
			}

			ctx := cmd.Context()
			mergedLogsCh := mergeLogs(ctx, fun.Map[<-chan core.LogLine](func(proc core.ProcStat) <-chan core.LogLine {
				return implLogs(ctx, proc)
			}, procs...))

			procIDs := fun.Map[core.PMID](func(proc core.ProcStat) core.PMID { return proc.ID }, procs...)
			return tui(ctx, dbb, cfg, mergedLogsCh, procIDs...)
		},
	}
	addFlagGenerics(cmd, filter, &names, &tags, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
