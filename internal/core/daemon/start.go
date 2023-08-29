package daemon

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
)

// Start - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (srv *daemonServer) Start(ctx context.Context, req *pb.ProcID) (*emptypb.Empty, error) {
	srv.ebus.Publish(ctx, eventbus.NewPublishProcStartRequest(req.GetId(), eventbus.EmitReasonByUser))

	return &emptypb.Empty{}, nil
}
