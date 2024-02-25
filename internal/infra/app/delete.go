package app

import (
	stdErrors "errors"
	"io/fs"
	"os"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
)

func removeFile(name string) error {
	if errRm := os.Remove(name); errRm != nil {
		if stdErrors.Is(errRm, fs.ErrNotExist) {
			return nil
		}

		return errors.Wrap(errRm, "remove file")
	}

	return nil
}

func removeLogFiles(proc core.Proc) error {
	if errRmStdout := removeFile(proc.StdoutFile); errRmStdout != nil {
		return errors.Wrap(errRmStdout, "remove stdout file %s", proc.StdoutFile)
	}

	if errRmStderr := removeFile(proc.StderrFile); errRmStderr != nil {
		return errors.Wrap(errRmStderr, "remove stderr file: %s", proc.StderrFile)
	}

	return nil
}

func (app App) delete(id core.PMID) error {
	deletedProc, errDelete := app.db.Delete(id)
	if errDelete != nil {
		return errors.Wrap(errDelete, "delete proc: %s", id)
	}

	if err := removeLogFiles(deletedProc); err != nil {
		return errors.Wrap(err, "delete proc: %s", id)
	}

	return nil
}

func (app App) Delete(ids ...core.PMID) error {
	for _, id := range ids {
		if err := app.delete(id); err != nil {
			return errors.Wrap(err, "server.delete: %s", id)
		}
	}

	return nil
}
