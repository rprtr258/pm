package daemon

import (
	"context"
	"errors"
	"net"
	"os"

	"go.uber.org/fx"
	"google.golang.org/grpc"

	"github.com/rprtr258/pm/api"
	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	db               db.Handle
	ebus             *eventbus.EventBus
	homeDir, logsDir string
}

var moduleServer = fx.Options(
	fx.Provide(newListener),
	fx.Provide(newServer),
	fx.Invoke(func(*grpc.Server) {}),
)

func newListener(lc fx.Lifecycle) (net.Listener, error) {
	sock, err := net.Listen("unix", core.SocketRPC)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return sock.Close()
		},
	})

	return sock, nil
}

func newServer(lc fx.Lifecycle, sock net.Listener, ebus *eventbus.EventBus, dbHandle db.Handle) *grpc.Server {
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
	})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info().Stringer("socket", sock.Addr()).Msg("daemon started")
			return srv.Serve(sock)
		},
		OnStop: func(context.Context) error {
			srv.GracefulStop()

			if errRm := os.Remove(core.SocketRPC); errRm != nil && !errors.Is(errRm, os.ErrNotExist) {
				return xerr.NewWM(errRm, "remove pid file", xerr.Fields{"file": _filePid})
			}

			return nil
		},
	})
	return srv
}
