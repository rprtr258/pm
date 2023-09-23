package daemon

import (
	"context"
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

func (s *Server) Delete(_ context.Context, id core.ProcID) error {
	deletedProc, errDelete := s.db.Delete(id)
	if errDelete != nil {
		return xerr.NewWM(errDelete, "delete proc", xerr.Fields{"proc_id": id})
	}

	if err := removeLogFiles(deletedProc); err != nil {
		return xerr.NewWM(err, "delete proc", xerr.Fields{"proc_id": id})
	}

	return nil
}
