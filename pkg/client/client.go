package client

import (
	"context"
	"io"
	"net"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/cli/log"
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

func (c Client) Create(ctx context.Context, opts *pb.CreateRequest) (string, error) {
	resp, err := c.client.Create(ctx, opts)
	if err != nil {
		return "", xerr.NewWM(err, "server.create")
	}

	return resp.GetId(), nil
}

func (c Client) List(ctx context.Context) (core.Procs, error) {
	resp, err := c.client.List(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, xerr.NewWM(err, "server.list")
	}

	return fun.SliceToMap[core.PMID, core.Proc](
		resp.GetProcesses(),
		func(proc *pb.Process) (core.PMID, core.Proc) {
			procID := core.PMID(proc.GetId().GetId())
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
		return core.NewStatusStopped()
	case status.GetRunning() != nil:
		stat := status.GetRunning()
		return core.NewStatusRunning(
			stat.GetStartTime().AsTime(),
			stat.GetCpu(),
			stat.GetMemory(),
		)
	default:
		return core.NewStatusInvalid()
	}
}

func (c Client) Delete(ctx context.Context, id string) error {
	if _, err := c.client.Delete(ctx, &pb.ProcID{Id: id}); err != nil {
		return xerr.NewWM(err, "server.delete", xerr.Fields{"pmid": id})
	}
	return nil
}

func (c Client) Start(ctx context.Context, id string) error {
	if _, err := c.client.Start(ctx, &pb.ProcID{Id: id}); err != nil {
		return xerr.NewWM(err, "server.start", xerr.Fields{"pmid": id})
	}

	return nil
}

func (c Client) Stop(ctx context.Context, id string) error {
	if _, err := c.client.Stop(ctx, &pb.ProcID{Id: id}); err != nil {
		return xerr.NewWM(err, "server.stop", xerr.Fields{"pmid": id})
	}

	return nil
}

func (c Client) Signal(ctx context.Context, signal syscall.Signal, id core.PMID) error {
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
		Id:     &pb.ProcID{Id: id.String()},
		Signal: apiSignal,
	}); err != nil {
		return xerr.NewWM(err, "server.signal", xerr.Fields{"pmid": id})
	}

	return nil
}

type WatcherEntry struct {
	Root    string
	Pattern string
}

type Status struct {
	Status  linuxprocess.Status
	Watches map[core.PMID]WatcherEntry
}

func (c Client) HealthCheck(ctx context.Context) (Status, error) {
	status, err := c.client.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "server.healthcheck")
	}

	watches := map[core.PMID]WatcherEntry{}
	for k, v := range status.GetWatches() {
		watches[core.PMID(k)] = WatcherEntry{
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

func (c Client) Subscribe(ctx context.Context, id core.PMID) (<-chan core.Proc, error) {
	resp, err := c.client.Subscribe(ctx, &pb.ProcID{
		Id: id.String(),
	})
	if err != nil {
		return nil, xerr.NewWM(err, "server.subscribe")
	}

	res := make(chan core.Proc)
	go func() {
		defer close(res)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				proc, err := resp.Recv()
				if err != nil {
					if err != io.EOF {
						// res.Err <- err
						log.Error().Err(err).Str("err_str", err.Error()).Msg("failed to receive proc")
					}
					return
				}

				res <- core.Proc{
					ID:         core.PMID(proc.GetId().GetId()),
					Name:       proc.GetName(),
					Tags:       proc.GetTags(),
					Command:    proc.GetCommand(),
					Args:       proc.GetArgs(),
					Cwd:        proc.GetCwd(),
					StdoutFile: proc.GetStdoutFile(),
					StderrFile: proc.GetStderrFile(),
					Watch:      fun.FromPtr(proc.Watch),
					Status:     mapPbStatus(proc.GetStatus()),
					Env:        proc.GetEnv(),
				}
			}
		}
	}()
	return res, nil
}
