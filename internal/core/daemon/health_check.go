package daemon

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"
)

func (*daemonServer) HealthCheck(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
