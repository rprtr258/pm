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

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
)

type Client struct {
	client pb.DaemonClient
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
		client: pb.NewDaemonClient(conn),
		conn:   conn,
	}, nil
}

func (c Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return xerr.NewWM(err, "close client")
	}

	return nil
}

func (c Client) Create(ctx context.Context, opts *pb.CreateRequest) (uint64, error) {
	resp, err := c.client.Create(ctx, opts)
	if err != nil {
		return 0, xerr.NewWM(err, "server.create")
	}

	return resp.GetId(), nil
}

func (c Client) List(ctx context.Context) (map[core.ProcID]core.Proc, error) {
	resp, err := c.client.List(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, xerr.NewWM(err, "server.list")
	}

	return lo.SliceToMap(
		resp.GetProcesses(),
		func(proc *pb.Process) (core.ProcID, core.Proc) {
			procID := proc.GetId()
			return procID, core.Proc{
				ID:         procID,
				Name:       proc.GetName(),
				Tags:       proc.GetTags(),
				Command:    proc.GetCommand(),
				Args:       proc.GetArgs(),
				Env:        proc.GetEnv(),
				Cwd:        proc.GetCwd(),
				StdoutFile: proc.GetStdoutFile(),
				StderrFile: proc.GetStderrFile(),
				Watch:      fun.FromPtr(proc.Watch),
				Status:     mapPbStatus(proc.GetStatus()),
			}
		},
	), nil
}

func mapPbStatus(status *pb.ProcessStatus) core.Status {
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

func (c Client) Delete(ctx context.Context, id uint64) error {
	if _, err := c.client.Delete(ctx, &pb.ProcID{Id: id}); err != nil {
		return xerr.NewWM(err, "server.delete", xerr.Fields{"proc_id": id})
	}
	return nil
}

func (c Client) Start(ctx context.Context, id uint64) error {
	if _, err := c.client.Start(ctx, &pb.ProcID{Id: id}); err != nil {
		return xerr.NewWM(err, "server.start", xerr.Fields{"proc_id": id})
	}

	return nil
}

func (c Client) Stop(ctx context.Context, id uint64) error {
	if _, err := c.client.Stop(ctx, &pb.ProcID{Id: id}); err != nil {
		return xerr.NewWM(err, "server.stop", xerr.Fields{"proc_id": id})
	}

	return nil
}

func (c Client) Signal(ctx context.Context, signal syscall.Signal, ids []uint64) error {
	var apiSignal pb.Signal
	switch signal { //nolint:exhaustive // other signals are not supported now
	case syscall.SIGTERM:
		apiSignal = pb.Signal_SIGNAL_SIGTERM
	case syscall.SIGKILL:
		apiSignal = pb.Signal_SIGNAL_SIGKILL
	default:
		return xerr.NewM("unknown signal", xerr.Fields{"signal": signal})
	}

	if _, err := c.client.Signal(ctx, &pb.SignalRequest{
		Ids:    ids,
		Signal: apiSignal,
	}); err != nil {
		return xerr.NewWM(err, "server.signal", xerr.Fields{"ids": ids})
	}

	return nil
}

func (c Client) HealthCheck(ctx context.Context) error {
	if _, err := c.client.HealthCheck(ctx, &emptypb.Empty{}); err != nil {
		return xerr.NewWM(err, "server.healthcheck")
	}

	return nil
}

type LogsIterator struct {
	Logs chan *pb.ProcsLogs
	Err  chan error
}

func (c Client) Logs(ctx context.Context, id core.ProcID) (LogsIterator, error) {
	res, err := c.client.Logs(ctx, &pb.ProcID{
		Id: id,
	})
	if err != nil {
		return fun.Zero[LogsIterator](), xerr.NewWM(err, "server.logs")
	}

	res2 := LogsIterator{
		Logs: make(chan *pb.ProcsLogs),
		Err:  make(chan error, 1),
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				res2.Err <- nil
				return
			default:
				line, err := res.Recv()
				if err != nil {
					res2.Err <- err
					return
				}

				res2.Logs <- line
			}
		}
	}()

	return res2, nil
}
