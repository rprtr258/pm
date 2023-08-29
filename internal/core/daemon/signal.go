package daemon

import (
	"context"
	"syscall"

	"github.com/rprtr258/xerr"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
)

// Signal - send signal processes to processes
func (srv *daemonServer) Signal(ctx context.Context, req *pb.SignalRequest) (*emptypb.Empty, error) {
	var signal syscall.Signal
	switch req.GetSignal() {
	case pb.Signal_SIGNAL_SIGTERM:
		signal = syscall.SIGTERM
	case pb.Signal_SIGNAL_SIGKILL:
		signal = syscall.SIGKILL
	case pb.Signal_SIGNAL_UNSPECIFIED:
		return nil, xerr.NewM("signal was not specified")
	default:
		return nil, xerr.NewM("unknown signal", xerr.Fields{"signal": req.GetSignal()})
	}

	srv.ebus.Publish(ctx, eventbus.NewPublishProcSignalRequest(signal, req.GetId().GetId()))

	return &emptypb.Empty{}, nil
}
