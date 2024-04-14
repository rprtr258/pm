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

var _cmdStop = func() *cobra.Command {
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "stop [name|tag|id]...",
		Short:             "stop process(es)",
		Aliases:           []string{"kill"},
		GroupID:           "inspection",
		ValidArgsFunction: completeArgGenericSelector,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			client, errList := app.New()
			if errList != nil {
				return errors.Wrapf(errList, "new grpc client")
			}

			list := client.List()

			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrapf(errLoadConfigs, "load configs")
				}

				names := fun.FilterMap[string](func(cfg core.RunConfig) fun.Option[string] {
					return cfg.Name
				}, configs...)

				list = list.
					Filter(func(proc core.Proc) bool {
						return fun.Contains(proc.Name, names...)
					})
			}

			procIDs := iter.Map(list.
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
				fmt.Println("nothing to stop")
				return nil
			}

			if err := client.Stop(procIDs...); err != nil {
				return errors.Wrapf(err, "client.stop")
			}

			return nil
		},
	}
	// &cli.BoolFlag{
	// 	Name:  "watch",
	// 	Usage: "stop watching for file changes",
	// },
	// &cli.BoolFlag{
	// 	Name:  "kill",
	// 	Usage: "kill process with SIGKILL instead of SIGINT",
	// },
	// &cli.DurationFlag{
	// 	Name:    "kill-timeout",
	// 	Aliases: []string{"k"},
	// 	Usage:   "delay before sending final SIGKILL signal to process",
	// },
	// &cli.BoolFlag{
	// 	Name:  "no-treekill",
	// 	Usage: "Only kill the main process, not detached children",
	// },
	// TODO: -i/... to confirm which procs will be stopped
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
