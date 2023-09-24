package client

import (
	"context"
	"net"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

type Client struct {
	client pb.DaemonClient
	conn   *grpc.ClientConn
}

func New() (Client, error) {
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
		return fun.Zero[Client](), xerr.NewWM(err, "connect to grpc server")
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

func (c Client) List(ctx context.Context) (core.Procs, error) {
	resp, err := c.client.List(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, xerr.NewWM(err, "server.list")
	}

	return fun.SliceToMap[core.ProcID, core.Proc](
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

func (c Client) Signal(ctx context.Context, signal syscall.Signal, id uint64) error {
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
		Id:     &pb.ProcID{Id: id},
		Signal: apiSignal,
	}); err != nil {
		return xerr.NewWM(err, "server.signal", xerr.Fields{"proc_id": id})
	}

	return nil
}

type WatcherEntry struct {
	Root    string
	Pattern string
}

type Status struct {
	Status  linuxprocess.Status
	Watches map[core.ProcID]WatcherEntry
}

func (c Client) HealthCheck(ctx context.Context) (Status, error) {
	status, err := c.client.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "server.healthcheck")
	}

	watches := map[core.ProcID]WatcherEntry{}
	for k, v := range status.GetWatches() {
		watches[k] = WatcherEntry{
			Root:    v.GetRoot(),
			Pattern: v.GetPattern(),
		}
	}

	return Status{
		Status: linuxprocess.Status{
			Args:       status.GetArgs(),
			Envs:       status.GetEnvs(),
			Executable: status.GetExecutable(),
			Cwd:        status.GetCwd(),
			Groups: fun.Map[int](status.GetGroups(), func(g int64) int {
				return int(g)
			}),
			PageSize:      int(status.GetPageSize()),
			Hostname:      status.GetHostname(),
			UserCacheDir:  status.GetUserCacheDir(),
			UserConfigDir: status.GetUserConfigDir(),
			UserHomeDir:   status.GetUserHomeDir(),
			PID:           int(status.GetPid()),
			PPID:          int(status.GetPpid()),
			UID:           int(status.GetUid()),
			EUID:          int(status.GetEuid()),
			GID:           int(status.GetGid()),
			EGID:          int(status.GetEgid()),
			PGID:          0,
			PGRP:          0,
			TID:           0,
		},
		Watches: watches,
	}, nil
}

type LogsIterator struct {
	Logs chan *pb.LogLine
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
		Logs: make(chan *pb.LogLine),
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
