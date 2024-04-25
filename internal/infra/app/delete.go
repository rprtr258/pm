package app

import (
	stdErrors "errors"
	"io/fs"
	"os"

	"go.uber.org/multierr"

	"github.com/rprtr258/pm/internal/core"
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

func Delete(app App, ids ...core.PMID) error {
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
