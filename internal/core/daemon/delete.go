package daemon

import (
	"context"
	"errors"
	"io/fs"
	"os"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/xerr"
	"google.golang.org/protobuf/types/known/emptypb"
)

func removeFile(name string) error {
	if _, errStat := os.Stat(name); errStat != nil {
		if errors.Is(errStat, fs.ErrNotExist) {
			return nil
		}
		return xerr.NewWM(errStat, "remove file, stat", xerr.Fields{"filename": name})
	}

	if errRm := os.Remove(name); errRm != nil {
		return xerr.NewWM(errRm, "remove file", xerr.Fields{"filename": name})
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

func (srv *daemonServer) Delete(_ context.Context, req *pb.ProcID) (*emptypb.Empty, error) {
	id := req.GetId()
	deletedProc, errDelete := srv.db.Delete(id)
	if errDelete != nil {
		return nil, xerr.NewWM(errDelete, "delete proc", xerr.Fields{"proc_id": id})
	}

	if err := removeLogFiles(deletedProc); err != nil {
		return nil, xerr.NewWM(err, "delete proc", xerr.Fields{"proc_id": id})
	}

	return &emptypb.Empty{}, nil
}
