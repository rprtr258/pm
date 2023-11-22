package app

import (
	"errors"
	"io/fs"
	"os"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
)

func removeFile(name string) error {
	if errRm := os.Remove(name); errRm != nil {
		if errors.Is(errRm, fs.ErrNotExist) {
			return nil
		}

		return xerr.NewWM(errRm, "remove file")
	}

	return nil
}

func removeLogFiles(proc core.Proc) error {
	if errRmStdout := removeFile(proc.StdoutFile); errRmStdout != nil {
		return xerr.NewWM(errRmStdout, "remove stdout file", xerr.Fields{"stdout_file": proc.StdoutFile})
	}

	if errRmStderr := removeFile(proc.StderrFile); errRmStderr != nil {
		return xerr.NewWM(errRmStderr, "remove stderr file", xerr.Fields{"stderr_file": proc.StderrFile})
	}

	return nil
}

func (app App) delete(id core.PMID) error {
	deletedProc, errDelete := app.db.Delete(id)
	if errDelete != nil {
		return xerr.NewWM(errDelete, "delete proc", xerr.Fields{"pmid": id})
	}

	if err := removeLogFiles(deletedProc); err != nil {
		return xerr.NewWM(err, "delete proc", xerr.Fields{"pmid": id})
	}

	return nil
}

func (app App) Delete(ids ...core.PMID) error {
	for _, id := range ids {
		if err := app.delete(id); err != nil {
			return xerr.NewWM(err, "server.delete", xerr.Fields{"pmid": id})
		}
	}

	return nil
}
