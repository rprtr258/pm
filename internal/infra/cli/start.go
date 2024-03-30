package cli

import (
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
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

			app, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrap(errNewApp, "new app")
			}

			list := app.List()

			if config == nil {
				procIDs := iter.Map(list.
					Filter(core.FilterFunc(
						core.WithGeneric(args...),
						core.WithIDs(ids...),
						core.WithNames(names...),
						core.WithTags(tags...),
					)),
					func(proc core.Proc) core.PMID {
						return proc.ID
					}).
					ToSlice()
				if len(procIDs) == 0 {
					fmt.Println("nothing to start")
					return nil
				}

				if err := app.Start(procIDs...); err != nil {
					return errors.Wrap(err, "client.start")
				}

				printIDs(procIDs...)

				return nil
			}

			configs, errLoadConfigs := core.LoadConfigs(*config)
			if errLoadConfigs != nil {
				return errors.Wrap(errLoadConfigs, "load configs: %s", *config)
			}

			procIDs := iter.Map(app.
				ListByRunConfigs(configs).
				Filter(core.FilterFunc(
					core.WithGeneric(args...),
					core.WithIDs(ids...),
					core.WithNames(names...),
					core.WithTags(tags...),
					core.WithAllIfNoFilters,
				)),
				func(proc core.Proc) core.PMID {
					return proc.ID
				}).
				ToSlice()
			if len(procIDs) == 0 {
				fmt.Println("nothing to start")
				return nil
			}

			if err := app.Start(procIDs...); err != nil {
				return errors.Wrap(err, "client.start")
			}

			printIDs(procIDs...)

			return nil
		},
	}
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
