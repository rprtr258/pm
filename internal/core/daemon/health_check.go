package daemon

import (
	"context"
	"os"
	"strings"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/rprtr258/fun"
	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/xerr"
)

func (*daemonServer) HealthCheck(context.Context, *emptypb.Empty) (*pb.Status, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, xerr.NewWM(err, "get executable")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, xerr.NewWM(err, "get cwd")
	}

	groups, err := os.Getgroups()
	if err != nil {
		return nil, xerr.NewWM(err, "get groups")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, xerr.NewWM(err, "get hostname")
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, xerr.NewWM(err, "get userCacheDir")
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, xerr.NewWM(err, "get userConfigDir")
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, xerr.NewWM(err, "get userHomeDir")
	}

	return &pb.Status{
		Args: os.Args,
		Envs: fun.SliceToMap[string, string](os.Environ(), func(v string) (string, string) {
			name, val, _ := strings.Cut(v, "=")
			return name, val
		}),
		Executable: executable,
		Cwd:        cwd,
		Groups: fun.Map[int64](groups, func(id int) int64 {
			return int64(id)
		}),
		PageSize:      int64(os.Getpagesize()),
		Hostname:      hostname,
		UserCacheDir:  userCacheDir,
		UserConfigDir: userConfigDir,
		UserHomeDir:   userHomeDir,
		Pid:           int64(os.Getpid()),
		Ppid:          int64(os.Getppid()),
		Uid:           int64(os.Getuid()),
		Euid:          int64(os.Geteuid()),
		Gid:           int64(os.Getgid()),
		Egid:          int64(os.Getegid()),
	}, nil
}
