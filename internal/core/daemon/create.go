package daemon

import (
	"context"

	"github.com/rprtr258/fun"
	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core/daemon/runner"
)

func (srv *daemonServer) Create(ctx context.Context, req *pb.CreateRequest) (*pb.ProcID, error) {
	// TODO: ARCH: move create out of runner, remove runner from server struct
	procID, err := srv.runner.Create(ctx, runner.CreateQuery{
		Name:       fun.FromPtr(req.Name),
		Cwd:        req.GetCwd(),
		Tags:       req.GetTags(),
		Command:    req.GetCommand(),
		Args:       req.GetArgs(),
		Watch:      fun.FromPtr(req.Watch),
		Env:        req.GetEnv(),
		StdoutFile: fun.FromPtr(req.StdoutFile),
		StderrFile: fun.FromPtr(req.StderrFile),
	})
	if err != nil {
		return nil, err
	}

	return &pb.ProcID{
		Id: procID,
	}, nil
}
