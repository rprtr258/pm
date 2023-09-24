package daemon

import (
	"context"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon"
	"github.com/rprtr258/pm/internal/core/daemon/watcher"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	srv *daemon.Server
}

func (s *daemonServer) HealthCheck(context.Context, *emptypb.Empty) (*pb.Status, error) {
	status, err := linuxprocess.GetSelfStatus()
	if err != nil {
		return nil, xerr.NewWM(err, "get proc status")
	}

	watches := map[core.ProcID]*pb.Watchplace{}
	iter.FromDict(s.srv.W.Watchplaces)(func(kv fun.Pair[core.ProcID, watcher.WatcherEntry]) bool {
		watches[kv.K] = &pb.Watchplace{
			Root:    kv.V.RootDir,
			Pattern: kv.V.Pattern.String(),
		}
		return true
	})

	return &pb.Status{
		Args:       status.Args,
		Envs:       status.Envs,
		Executable: status.Executable,
		Cwd:        status.Cwd,
		Groups: fun.Map[int64](status.Groups, func(id int) int64 {
			return int64(id)
		}),
		PageSize:      int64(status.PageSize),
		Hostname:      status.Hostname,
		UserCacheDir:  status.UserCacheDir,
		UserConfigDir: status.UserConfigDir,
		UserHomeDir:   status.UserHomeDir,
		Pid:           int64(status.PID),
		Ppid:          int64(status.PPID),
		Uid:           int64(status.UID),
		Euid:          int64(status.EUID),
		Gid:           int64(status.GID),
		Egid:          int64(status.EGID),
		Watches:       watches,
	}, nil
}

func (s *daemonServer) Create(_ context.Context, req *pb.CreateRequest) (*pb.ProcID, error) {
	procID, err := s.srv.Create(
		req.GetCommand(),
		req.GetArgs(),
		fun.FromPtr(req.Name),
		req.GetCwd(),
		req.GetTags(),
		req.GetEnv(),
		fun.FromPtr(req.Watch),
		fun.FromPtr(req.StdoutFile),
		fun.FromPtr(req.StderrFile),
	)
	if err != nil {
		return nil, err
	}

	return &pb.ProcID{
		Id: procID,
	}, nil
}

func (s *daemonServer) Start(ctx context.Context, req *pb.ProcID) (*emptypb.Empty, error) {
	s.srv.Start(ctx, req.GetId())
	return &emptypb.Empty{}, nil
}

func (s *daemonServer) Delete(ctx context.Context, req *pb.ProcID) (*emptypb.Empty, error) {
	if err := s.srv.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

//nolint:exhaustruct // can't return api.isProcessStatus_Status
func mapStatus(status core.Status) *pb.ProcessStatus {
	switch status.Status {
	case core.StatusInvalid:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Invalid{}}
	case core.StatusCreated:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Created{}}
	case core.StatusStopped:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Stopped{
			Stopped: &pb.StoppedProcessStatus{
				ExitCode:  int64(status.ExitCode),
				StoppedAt: timestamppb.New(status.StoppedAt),
			},
		}}
	case core.StatusRunning:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Running{
			Running: &pb.RunningProcessStatus{
				Pid:       int64(status.Pid),
				StartTime: timestamppb.New(status.StartTime),
				// TODO: get from /proc/PID/stat
				Cpu:    status.CPU,
				Memory: status.Memory,
			},
		}}
	default:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Invalid{}}
	}
}

func (s *daemonServer) List(ctx context.Context, _ *emptypb.Empty) (*pb.ProcessesList, error) {
	return &pb.ProcessesList{
		Processes: fun.MapToSlice(
			s.srv.List(ctx),
			func(id core.ProcID, proc core.Proc) *pb.Process {
				return &pb.Process{
					Id:         id,
					Name:       proc.Name,
					Tags:       proc.Tags,
					Command:    proc.Command,
					Args:       proc.Args,
					Cwd:        proc.Cwd,
					Env:        proc.Env,
					StdoutFile: proc.StdoutFile,
					StderrFile: proc.StderrFile,
					Watch:      proc.Watch.Ptr(),
					Status:     mapStatus(proc.Status),
				}
			},
		),
	}, nil
}

func (s *daemonServer) Stop(ctx context.Context, req *pb.ProcID) (*emptypb.Empty, error) {
	s.srv.Stop(ctx, req.GetId())

	return &emptypb.Empty{}, nil
}

func (s *daemonServer) Signal(ctx context.Context, req *pb.SignalRequest) (*emptypb.Empty, error) {
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

	s.srv.Signal(ctx, req.GetId().GetId(), signal)

	return &emptypb.Empty{}, nil
}

func (s *daemonServer) Logs(req *pb.ProcID, stream pb.Daemon_LogsServer) error {
	ch, err := s.srv.Logs(stream.Context(), req.GetId())
	if err != nil {
		return err
	}

	for {
		select {
		case line := <-ch:
			if errSend := stream.Send(&pb.LogLine{
				Id:   line.ID,
				Name: line.Name,
				Line: line.Line,
				At:   timestamppb.New(line.At),
				Type: lo.Switch[core.LogType, pb.LogLine_Type](line.Type).
					Case(core.LogTypeStdout, pb.LogLine_TYPE_STDOUT).
					Case(core.LogTypeStderr, pb.LogLine_TYPE_STDERR).
					Default(pb.LogLine_TYPE_UNSPECIFIED),
			}); errSend != nil {
				return xerr.NewWM(errSend, "send log lines", xerr.Fields{
					"procID": req.GetId(),
				})
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}
