package cli

import (
	stdErrors "errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
)

func removeFile(name string) error {
	if errRm := os.Remove(name); errRm != nil {
		if stdErrors.Is(errRm, fs.ErrNotExist) {
			return nil
		}

		return errors.Wrapf(errRm, "remove file")
	}

	return nil
}

func ImplDelete(app app.App, ids ...core.PMID) error {
	var merr error
	for _, id := range ids {
		if err := func() error {
			proc, errDelete := app.DB.Delete(id)
			if errDelete != nil {
				return errors.Wrapf(errDelete, "delete proc: %s", id)
			}

			// remove log files
			if err := multierr.Combine(
				errors.Wrapf(removeFile(proc.StdoutFile), "remove stdout file %s", proc.StdoutFile),
				errors.Wrapf(removeFile(proc.StderrFile), "remove stderr file: %s", proc.StderrFile),
			); err != nil {
				return err
			}

			return nil
		}(); err != nil {
			multierr.AppendInto(&merr, errors.Wrapf(err, "server.delete: pmid=%s", id))
		}
	}
	return merr
}

var _cmdDelete = func() *cobra.Command {
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "delete [name|tag|id]...",
		Short:             "stop and remove process(es)",
		GroupID:           "management",
		ValidArgsFunction: completeArgGenericSelector,
		Aliases:           []string{"del", "rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			appp, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrapf(errNewApp, "new app")
			}

			list := appp.List()
			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrapf(errLoadConfigs, "load configs: %s", *config)
				}

				procNames := fun.FilterMap[string](func(cfg core.RunConfig) (string, bool) {
					return cfg.Name.Unpack()
				}, configs...)

				list = list.
					Filter(func(proc core.Proc) bool {
						return fun.Contains(proc.Name, procNames...)
					}).
					Filter(core.FilterFunc(
						core.WithGeneric(args...),
						core.WithIDs(ids...),
						core.WithNames(names...),
						core.WithTags(tags...),
						core.WithAllIfNoFilters,
					))
			} else {
				list = list.
					Filter(core.FilterFunc(
						core.WithGeneric(args...),
						core.WithIDs(ids...),
						core.WithNames(names...),
						core.WithTags(tags...),
					))
			}

			procIDs := iter.Map(list,
				func(proc core.Proc) core.PMID {
					return proc.ID
				}).
				ToSlice()
			if len(procIDs) == 0 {
				fmt.Println("Nothing to delete, leaving")
				return nil
			}

			if err := appp.Stop(procIDs...); err != nil {
				return errors.Wrapf(err, "stop")
			}

			if err := ImplDelete(appp, procIDs...); err != nil {
				return errors.Wrapf(err, "delete")
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
