package cli

import (
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/errors"
)

var _cmdStop = func() *cobra.Command {
	const filter = filterRunning
	var names, ids, tags []string
	var config string
	var interactive bool
	cmd := &cobra.Command{
		Use:               "stop [name|tag|id]...",
		Short:             "stop process(es)",
		Aliases:           []string{"kill"},
		ValidArgsFunction: completeArgGenericSelector(filter),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			list := listProcs(dbb)
			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrapf(errLoadConfigs, "load configs")
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
					return ps.Status != core.StatusStopped &&
						filterFunc(ps.Proc) &&
						(!interactive || confirmProc(ps, "stop"))
				}).
				Slice()
			if len(procs) == 0 {
				fmt.Println("nothing to stop")
				return nil
			}

			procIDs := fun.Map[core.PMID](func(proc core.ProcStat) core.PMID { return proc.ID }, procs...)
			if err := implStop(dbb, procIDs...); err != nil {
				return errors.Wrapf(err, "client.stop")
			}

			printProcs(procs...)

			return nil
		},
	}
	addFlagInteractive(cmd, &interactive)
	addFlagGenerics(cmd, filter, &names, &tags, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
