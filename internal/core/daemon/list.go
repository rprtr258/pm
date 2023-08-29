package daemon

import (
	"context"

	"github.com/rprtr258/fun"
	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

func (srv *daemonServer) List(_ context.Context, _ *emptypb.Empty) (*pb.ProcessesList, error) {
	return &pb.ProcessesList{
		Processes: fun.MapToSlice(
			srv.db.GetProcs(core.WithAllIfNoFilters),
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
