package cli

import (
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/errors"
)

var _cmdStart = func() *cobra.Command {
	const filter = filterStopped
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "start [name|tag|id]...",
		Short:             "start already added process(es)",
		GroupID:           "management",
		ValidArgsFunction: completeArgGenericSelector(filter),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			var filterFunc func(core.Proc) bool
			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrapf(errLoadConfigs, "load configs: %s", *config)
				}

				procNames := fun.Map[string](func(cfg core.RunConfig) string {
					return cfg.Name
				}, configs...)

				ff := core.FilterFunc(
					core.WithGeneric(args...),
					core.WithIDs(ids...),
					core.WithNames(names...),
					core.WithTags(tags...),
					core.WithAllIfNoFilters,
				)
				filterFunc = func(proc core.Proc) bool {
					return fun.Contains(proc.Name, procNames...) && ff(proc)
				}
			} else {
				filterFunc = core.FilterFunc(
					core.WithGeneric(args...),
					core.WithIDs(ids...),
					core.WithNames(names...),
					core.WithTags(tags...),
				)
			}

			procs := listProcs(dbb).
				Filter(func(ps core.ProcStat) bool { return filterFunc(ps.Proc) }).
				Slice()
			if len(procs) == 0 {
				fmt.Println("nothing to start")
				return nil
			}

			procIDs := fun.Map[core.PMID](
				func(proc core.ProcStat) core.PMID {
					return proc.ID
				}, procs...)
			if err := implStart(dbb, procIDs...); err != nil {
				return err
			}

			printProcs(procs...)

			return nil
		},
	}
	addFlagGenerics(cmd, filter, &names, &tags, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
