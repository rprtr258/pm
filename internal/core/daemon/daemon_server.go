package daemon

import (
	"github.com/rprtr258/pm/api"
	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/runner"
	"github.com/rprtr258/pm/internal/infra/db"
	"google.golang.org/grpc"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	db               db.Handle
	ebus             *eventbus.EventBus
	homeDir, logsDir string
	runner           runner.Runner // TODO: remove, used only for create
}

func newServer(
	dbHandle db.Handle,
	ebus *eventbus.EventBus,
	pmRunner runner.Runner,
) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryLoggerInterceptor),
		grpc.ChainStreamInterceptor(streamLoggerInterceptor),
	)
	api.RegisterDaemonServer(srv, &daemonServer{
		UnimplementedDaemonServer: api.UnimplementedDaemonServer{},
		db:                        dbHandle,
		homeDir:                   core.DirHome,
		logsDir:                   _dirProcsLogs,
		ebus:                      ebus,
		runner:                    pmRunner,
	})
	return srv
}
