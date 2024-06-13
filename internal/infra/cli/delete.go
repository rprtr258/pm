package cli

import (
	stdErrors "errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/spf13/cobra"

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

func removeFileGlob(glob string) error {
	names, err := filepath.Glob(glob)
	if err != nil {
		// ignore
		return nil
	}

	for _, name := range names {
		if errRm := os.Remove(name); errRm != nil {
			if stdErrors.Is(errRm, fs.ErrNotExist) {
				return nil
			}

			return errors.Wrapf(errRm, "remove file %s", name)
		}
	}

	return nil
}

func ImplDelete(appp app.App, ids ...core.PMID) error {
	return errors.Combine(fun.Map[error](func(id core.PMID) error {
		return errors.Wrapf(func() error {
			proc, errDelete := appp.DB.Delete(id)
			if errDelete != nil {
				return errors.Wrapf(errDelete, "delete proc: %s", id)
			}

			fmt.Println(proc.Name)

			// remove log files
			return errors.Combine(
				errors.Wrapf(removeFileGlob(filepath.Join(appp.DirLogs, proc.ID.String()+"*")), "remove logrotation files"),
				errors.Wrapf(removeFile(proc.StdoutFile), "remove stdout file %s", proc.StdoutFile),
				errors.Wrapf(removeFile(proc.StderrFile), "remove stderr file: %s", proc.StderrFile),
			)
		}(), "server.delete: pmid=%s", id)
	}, ids...)...)
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
				filterFunc = func(p core.Proc) bool {
					return fun.Contains(p.Name, procNames...) && ff(p)
				}
			} else {
				filterFunc = core.FilterFunc(
					core.WithGeneric(args...),
					core.WithIDs(ids...),
					core.WithNames(names...),
					core.WithTags(tags...),
				)
			}

			list := appp.
				List().
				Filter(func(ps core.ProcStat) bool { return filterFunc(ps.Proc) })
			procIDs := iter.Map(list,
				func(proc core.ProcStat) core.PMID {
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
