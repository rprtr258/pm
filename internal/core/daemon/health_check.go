package daemon

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/rprtr258/fun"
	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
	"github.com/rprtr258/xerr"
)

func (*daemonServer) HealthCheck(context.Context, *emptypb.Empty) (*pb.Status, error) {
	status, err := linuxprocess.HealthCheck()
	if err != nil {
		return nil, xerr.NewWM(err, "get proc status")
	}

	return &pb.Status{
		Args:       status.Args,
		Envs:       status.Envs,
		Executable: status.Executable,
		Cwd:        status.Cwd,
		Groups: fun.Map[int64](status.Groups, func(id int) int64 {
			return int64(id)
		}),
		PageSize:      int64(status.PageSize),
		Hostname:      status.Hostname,
		UserCacheDir:  status.UserCacheDir,
		UserConfigDir: status.UserConfigDir,
		UserHomeDir:   status.UserHomeDir,
		Pid:           int64(status.PID),
		Ppid:          int64(status.PPID),
		Uid:           int64(status.UID),
		Euid:          int64(status.EUID),
		Gid:           int64(status.GID),
		Egid:          int64(status.EGID),
	}, nil
}
