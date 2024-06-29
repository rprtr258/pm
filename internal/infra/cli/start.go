package cli

import (
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
)

var _cmdStart = func() *cobra.Command {
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "start [name|tag|id]...",
		Short:             "start already added process(es)",
		GroupID:           "management",
		ValidArgsFunction: completeArgGenericSelector,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			appp, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrapf(errNewApp, "new app")
			}

			var filterFunc func(core.Proc) bool
			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrapf(errLoadConfigs, "load configs: %s", *config)
				}

				procNames := fun.FilterMap[string](func(cfg core.RunConfig) (string, bool) {
					return cfg.Name.Unpack()
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

			procs := listProcs(appp.DB).
				Filter(func(ps core.ProcStat) bool { return filterFunc(ps.Proc) }).
				ToSlice()
			if len(procs) == 0 {
				fmt.Println("nothing to start")
				return nil
			}

			procIDs := fun.Map[core.PMID](
				func(proc core.ProcStat) core.PMID {
					return proc.ID
				}, procs...)
			if err := appp.Start(procIDs...); err != nil {
				return err
			}

			printProcs(procs...)

			return nil
		},
	}
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
