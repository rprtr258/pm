package cli

import (
	stdErrors "errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/pm/internal/errors"
)

func removeFile(name string) error {
	if errRm := syscall.Unlink(name); errRm != nil {
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
		//nolint:nilerr // ignore
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

func implDelete(db db.Handle, dirLogs string, ids ...core.PMID) error {
	return errors.Combine(fun.Map[error](func(id core.PMID) error {
		return errors.Wrapf(func() error {
			proc, errDelete := db.Delete(id)
			if errDelete != nil {
				return errors.Wrapf(errDelete, "delete proc: %s", id)
			}

			fmt.Println(proc.Name)

			// remove log files
			return errors.Combine(
				errors.Wrapf(removeFileGlob(filepath.Join(dirLogs, proc.ID.String()+"*")), "remove logrotation files"),
				errors.Wrapf(removeFile(proc.StdoutFile), "remove stdout file %s", proc.StdoutFile),
				errors.Wrapf(removeFile(proc.StderrFile), "remove stderr file: %s", proc.StderrFile),
			)
		}(), "server.delete: pmid=%s", id)
	}, ids...)...)
}

var _cmdDelete = func() *cobra.Command {
	const filter = filterAll
	var names, ids, tags []string
	var config string
	var interactive bool
	cmd := &cobra.Command{
		Use:               "delete [name|tag|id]...",
		Short:             "stop and remove process(es)",
		GroupID:           "management",
		ValidArgsFunction: completeArgGenericSelector(filter),
		Aliases:           []string{"del", "rm"},
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

			procIDs := slices.Collect(listProcs(dbb).
				Filter(func(ps core.ProcStat) bool {
					return filterFunc(ps.Proc) &&
						(!interactive || confirmProc(ps, "delete"))
				}).
				IDs())
			if len(procIDs) == 0 {
				fmt.Println("Nothing to delete, leaving")
				return nil
			}

			if err := implStop(dbb, procIDs...); err != nil {
				return errors.Wrapf(err, "stop")
			}

			if err := implDelete(dbb, cfg.DirLogs, procIDs...); err != nil {
				return errors.Wrapf(err, "delete")
			}

			return nil
		},
	}
	addFlagInteractive(cmd, &interactive)
	addFlagGenerics(cmd, filter, &names, &tags, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
