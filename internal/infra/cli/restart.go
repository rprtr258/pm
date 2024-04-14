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
					fmt.Println("nothing to restart")
					return nil
				}

				if err := app.Stop(procIDs...); err != nil {
					return errors.Wrapf(err, "client.stop")
				}

				if errStart := app.Start(procIDs...); errStart != nil {
					return errors.Wrapf(errStart, "client.start")
				}

				return nil
			}

			configFile := *config

			configs, errLoadConfigs := core.LoadConfigs(configFile)
			if errLoadConfigs != nil {
				return errors.Wrapf(errLoadConfigs, "load configs from %s", configFile)
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

			if errStop := app.Stop(procIDs...); errStop != nil {
				return errors.Wrapf(errStop, "client.stop")
			}

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
