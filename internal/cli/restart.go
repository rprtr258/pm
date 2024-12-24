package cli

import (
	"fmt"
	"slices"

	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/errors"
)

var _cmdRestart = func() *cobra.Command {
	const filter = filterRunning
	var names, ids, tags []string
	var config string
	var interactive bool
	cmd := &cobra.Command{
		Use:               "restart [name|tag|id]...",
		Short:             "restart already added process(es)",
		GroupID:           "management",
		ValidArgsFunction: completeArgGenericSelector(filter),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			var filterFunc func(core.Proc) bool
			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrapf(errLoadConfigs, "load configs from %s", *config)
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

			procIDs := slices.Collect(listProcs(dbb).
				Filter(func(ps core.ProcStat) bool {
					return filterFunc(ps.Proc) &&
						(!interactive || confirmProc(ps, "restart"))
				}).
				IDs())
			if len(procIDs) == 0 {
				fmt.Println("nothing to restart")
				return nil
			}

			if errStop := implStop(dbb, procIDs...); errStop != nil {
				return errors.Wrapf(errStop, "client.stop")
			}

			if errStart := implStart(dbb, procIDs...); errStart != nil {
				return errors.Wrapf(errStart, "client.start")
			}

			return nil
		},
	}
	addFlagInteractive(cmd, &interactive)
	addFlagNames(cmd, &names, filter)
	addFlagTags(cmd, &tags, filter)
	addFlagIDs(cmd, &ids, filter)
	addFlagConfig(cmd, &config)
	return cmd
}()
