package daemon

import (
	"context"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (srv *daemonServer) Stop(ctx context.Context, req *pb.ProcID) (*emptypb.Empty, error) {
	srv.ebus.Publish(ctx, eventbus.NewPublishProcStopRequest(req.GetId(), eventbus.EmitReasonByUser))

	return &emptypb.Empty{}, nil
}
