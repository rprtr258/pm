package cli

import (
	"fmt"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
)

var _cmdRestart = func() *cobra.Command {
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "restart [name|tag|id]...",
		Short:             "restart already added process(es)",
		GroupID:           "management",
		ValidArgsFunction: completeArgGenericSelector,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			app, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrapf(errNewApp, "new app")
			}

			var filterFunc func(core.Proc) bool
			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrapf(errLoadConfigs, "load configs from %s", *config)
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

			list := app.
				List().
				Filter(func(ps core.ProcStat) bool { return filterFunc(ps.Proc) })
			procIDs := iter.Map(list,
				func(proc core.ProcStat) core.PMID {
					return proc.ID
				}).
				ToSlice()
			if len(procIDs) == 0 {
				fmt.Println("nothing to restart")
				return nil
			}

			if errStop := app.Stop(procIDs...); errStop != nil {
				return errors.Wrapf(errStop, "client.stop")
			}

			time.Sleep(3 * time.Second) // TODO: wait for killing

			if errStart := app.Start(procIDs...); errStart != nil {
				return errors.Wrapf(errStart, "client.start")
			}

			return nil
		},
	}
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
