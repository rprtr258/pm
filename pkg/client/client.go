package client

import (
	"context"
	"net"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
)

type Client struct {
	client api.DaemonClient
	conn   *grpc.ClientConn
}

func NewGrpcClient() (Client, error) {
	conn, err := grpc.Dial(
		core.SocketRPC,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			conn, err := net.Dial("unix", addr)
			if err != nil {
				return nil, xerr.NewWM(err, "dial unix socket",
					xerr.Fields{"addr": addr})
			}

			return conn, nil
		}),
	)
	if err != nil {
		return Client{}, xerr.NewWM(err, "connect to grpc server")
	}

	return Client{
		client: api.NewDaemonClient(conn),
		conn:   conn,
	}, nil
}

func (c Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return xerr.NewWM(err, "close client")
	}

	return nil
}

func (c Client) Create(ctx context.Context, opts []*api.ProcessOptions) ([]uint64, error) {
	resp, err := c.client.Create(ctx, &api.CreateRequest{Options: opts})
	if err != nil {
		return nil, xerr.NewWM(err, "server.create")
	}

	return fun.Map(resp.GetIds(), func(procID *api.ProcessID) uint64 {
		return procID.Id
	}), nil
}

func (c Client) List(ctx context.Context) (map[core.ProcID]core.ProcData, error) {
	resp, err := c.client.List(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, xerr.NewWM(err, "server.list")
	}

	return lo.SliceToMap(
		resp.GetProcesses(),
		func(proc *api.Process) (core.ProcID, core.ProcData) {
			procID := core.ProcID(proc.GetId().GetId())
			return procID, core.ProcData{
				ProcID:  procID,
				Name:    proc.GetName(),
				Command: proc.GetCommand(),
				Args:    proc.GetArgs(),
				Status:  mapPbStatus(proc.GetStatus()),
				Tags:    proc.GetTags(),
				Cwd:     proc.GetCwd(),
				Watch:   nil,
			}
		},
	), nil
}

func mapPbStatus(status *api.ProcessStatus) core.Status {
	switch {
	case status.GetInvalid() != nil:
		return core.NewStatusInvalid()
	case status.GetCreated() != nil:
		return core.NewStatusCreated()
	case status.GetStopped() != nil:
		return core.NewStatusStopped(int(status.GetStopped().GetExitCode()))
	case status.GetRunning() != nil:
		stat := status.GetRunning()
		return core.NewStatusRunning(
			stat.GetStartTime().AsTime(),
			int(stat.GetPid()),
			stat.GetCpu(),
			stat.GetMemory(),
		)
	default:
		return core.NewStatusInvalid()
	}
}

func (c Client) Delete(ctx context.Context, ids []uint64) error {
	if _, err := c.client.Delete(ctx, mapIDs(ids)); err != nil {
		return xerr.NewWM(err, "server.delete", xerr.Fields{"ids": ids})
	}
	return nil
}

func (c Client) Start(ctx context.Context, ids []uint64) error {
	if _, err := c.client.Start(ctx, mapIDs(ids)); err != nil {
		return xerr.NewWM(err, "server.start", xerr.Fields{"ids": ids})
	}

	return nil
}

func (c Client) Stop(ctx context.Context, ids []uint64) ([]uint64, error) {
	resIDs, err := c.client.Stop(ctx, &api.IDs{
		Ids: fun.Map(ids, mapProcID),
	})
	rawIDs := lo.Map(resIDs.GetIds(), func(procID *api.ProcessID, _ int) uint64 {
		return procID.GetId()
	})
	if err != nil {
		return rawIDs, xerr.NewWM(err, "server.stop", xerr.Fields{"ids": ids})
	}

	return rawIDs, nil
}

func (c Client) Signal(ctx context.Context, signal syscall.Signal, ids []uint64) error {
	var apiSignal api.Signal
	switch signal { //nolint:exhaustive // other signals are not supported now
	case syscall.SIGTERM:
		apiSignal = api.Signal_SIGNAL_SIGTERM
	case syscall.SIGKILL:
		apiSignal = api.Signal_SIGNAL_SIGKILL
	default:
		return xerr.NewM("unknown signal", xerr.Fields{"signal": signal})
	}

	if _, err := c.client.Signal(ctx, &api.SignalRequest{
		Ids:    fun.Map(ids, mapProcID),
		Signal: apiSignal,
	}); err != nil {
		return xerr.NewWM(err, "server.signal", xerr.Fields{"ids": ids})
	}

	return nil
}

func mapProcID(id uint64) *api.ProcessID {
	return &api.ProcessID{
		Id: id,
	}
}

func mapIDs(ids []uint64) *api.IDs {
	return &api.IDs{
		Ids: fun.Map(ids, mapProcID),
	}
}

func (c Client) HealthCheck(ctx context.Context) error {
	if _, err := c.client.HealthCheck(ctx, &emptypb.Empty{}); err != nil {
		return xerr.NewW(err)
	}
	return nil
}
