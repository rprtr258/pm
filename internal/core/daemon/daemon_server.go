package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/go-faster/tail"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/runner"
	"github.com/rprtr258/pm/internal/core/daemon/watcher"
	"github.com/rprtr258/pm/internal/infra/db"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	db               db.Handle
	watcher          watcher.Watcher
	runner           runner.Runner
	homeDir, logsDir string
}

// Start - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (srv *daemonServer) Start(ctx context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	ids := lo.Map(req.GetIds(), func(id uint64, _ int) core.ProcID {
		return core.ProcID(id)
	})

	if errStart := srv.runner.Start(ctx, ids...); errStart != nil {
		return nil, errStart
	}

	return &emptypb.Empty{}, nil
}

// Signal - send signal processes to processes
func (srv *daemonServer) Signal(_ context.Context, req *pb.SignalRequest) (*emptypb.Empty, error) {
	procsToStop := lo.Map(req.GetIds(), func(id uint64, _ int) core.ProcID {
		return core.ProcID(id)
	})

	procsWeHaveAmongRequested := srv.db.GetProcs(procsToStop)

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

	var merr error
	for _, proc := range procsWeHaveAmongRequested {
		xerr.AppendInto(&merr, srv.signal(proc, signal))
	}

	return &emptypb.Empty{}, merr
}

func (srv *daemonServer) Stop(ctx context.Context, req *pb.IDs) (*pb.IDs, error) {
	ids := lo.Map(req.GetIds(), func(id uint64, _ int) core.ProcID {
		return core.ProcID(id)
	})

	stoppedIDs, err := srv.runner.Stop(ctx, ids...)

	return &pb.IDs{
		Ids: lo.Map(stoppedIDs, func(id core.ProcID, _ int) uint64 {
			return uint64(id)
		}),
	}, err
}

func (srv *daemonServer) signal(proc core.Proc, signal syscall.Signal) error {
	if proc.Status.Status != core.StatusRunning {
		slog.Info("tried to send signal to non-running process",
			"proc", proc,
			"signal", signal,
		)
		return nil
	}

	process, errFindProc := os.FindProcess(proc.Status.Pid)
	if errFindProc != nil {
		return xerr.NewWM(errFindProc, "getting process by pid failed", xerr.Fields{
			"pid":    proc.Status.Pid,
			"signal": signal,
		})
	}

	if errKill := syscall.Kill(-process.Pid, signal); errKill != nil {
		switch {
		case errors.Is(errKill, os.ErrProcessDone):
			slog.Warn("tried to send signal to process which is done",
				"proc", proc,
				"signal", signal,
			)
		case errors.Is(errKill, syscall.ESRCH): // no such process
			slog.Warn("tried to send signal to process which doesn't exist",
				"proc", proc,
				"signal", signal,
			)
		default:
			return xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": process.Pid})
		}
	}

	return nil
}

func (srv *daemonServer) Create(ctx context.Context, req *pb.CreateRequest) (*pb.IDs, error) {
	procIDs, err := srv.runner.Create(ctx, lo.Map(req.GetOptions(), func(opts *pb.ProcessOptions, _ int) runner.CreateQuery {
		return runner.CreateQuery{
			Name:       fun.FromPtr(opts.Name),
			Cwd:        opts.GetCwd(),
			Tags:       opts.GetTags(),
			Command:    opts.GetCommand(),
			Args:       opts.GetArgs(),
			Watch:      fun.FromPtr(opts.Watch),
			Env:        opts.GetEnv(),
			StdoutFile: fun.FromPtr(opts.StdoutFile),
			StderrFile: fun.FromPtr(opts.StderrFile),
		}
	})...)
	if err != nil {
		return nil, err
	}

	return &pb.IDs{
		Ids: lo.Map(
			procIDs,
			func(procID core.ProcID, _ int) uint64 {
				return uint64(procID)
			},
		),
	}, nil
}

func (srv *daemonServer) List(ctx context.Context, _ *emptypb.Empty) (*pb.ProcessesList, error) {
	// TODO: update statuses here also
	list := srv.db.List()

	return &pb.ProcessesList{
		Processes: lo.MapToSlice(list, func(id core.ProcID, proc core.Proc) *pb.Process {
			return &pb.Process{
				Id:      uint64(id),
				Status:  mapStatus(proc.Status),
				Name:    proc.Name,
				Cwd:     proc.Cwd,
				Tags:    proc.Tags,
				Command: proc.Command,
				Args:    proc.Args,
				Watch:   proc.Watch.Ptr(),
				// TODO: fill with dirs if nil
				// StdoutFile: proc.StdoutFile,
				// StderrFile: proc.StderrFile,
			}
		}),
	}, nil
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
				// Cpu:       status.CPU,
				// Memory:    status.Memory,
			},
		}}
	default:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Invalid{}}
	}
}

func (srv *daemonServer) Delete(ctx context.Context, r *pb.IDs) (*emptypb.Empty, error) {
	ids := r.GetIds()
	if errDelete := srv.db.Delete(ids); errDelete != nil {
		return nil, xerr.NewWM(errDelete, "delete proc", xerr.Fields{"procIDs": ids})
	}

	var merr error
	for _, procID := range ids {
		if err := removeLogFiles(procID); err != nil {
			xerr.AppendInto(&merr, xerr.NewWM(err, "delete proc", xerr.Fields{"procID": procID}))
		}
	}

	return &emptypb.Empty{}, merr
}

func removeLogFiles(procID uint64) error {
	stdoutFilename := filepath.Join(_dirProcsLogs, fmt.Sprintf("%d.stdout", procID))
	if errRmStdout := removeFile(stdoutFilename); errRmStdout != nil {
		return errRmStdout
	}

	stderrFilename := filepath.Join(_dirProcsLogs, fmt.Sprintf("%d.stderr", procID))
	if errRmStderr := removeFile(stderrFilename); errRmStderr != nil {
		return errRmStderr
	}

	return nil
}

func removeFile(name string) error {
	if _, errStat := os.Stat(name); errStat != nil {
		if errors.Is(errStat, fs.ErrNotExist) {
			return nil
		}
		return xerr.NewWM(errStat, "remove file, stat", xerr.Fields{"filename": name})
	}

	if errRm := os.Remove(name); errRm != nil {
		return xerr.NewWM(errRm, "remove file", xerr.Fields{"filename": name})
	}

	return nil
}

func (srv *daemonServer) HealthCheck(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

const _procLogsBufferSize = 128 * 1024 // 128 kb

func (srv *daemonServer) Logs(req *pb.IDs, stream pb.Daemon_LogsServer) error {
	// can't get incoming query in interceptor, so logging here also
	slog.Info("Logs method called",
		slog.Any("ids", req.GetIds()),
	)

	procs := srv.db.List() // TODO: filter by ids
	done := make(chan struct{})

	var wgGlobal sync.WaitGroup
	for _, id := range req.GetIds() {
		proc, ok := procs[core.ProcID(id)]
		if !ok {
			slog.Info("tried to log unknown process", "procID", id)
			continue
		}

		wgGlobal.Add(2)
		go func(id core.ProcID) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var wgLocal sync.WaitGroup

			logLinesCh := make(chan *pb.LogLine)

			wgLocal.Add(1)

			stdoutTailer := tail.File(proc.StdoutFile, tail.Config{
				Follow:        true,
				BufferSize:    _procLogsBufferSize,
				NotifyTimeout: time.Minute,
				Location:      &tail.Location{Whence: io.SeekEnd, Offset: -1000}, // TODO: limit offset by file size as with daemon logs
			})
			go func() {
				if err := stdoutTailer.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
					select {
					case <-ctx.Done():
						wgGlobal.Done()
						return ctx.Err()
					case logLinesCh <- &pb.LogLine{
						Line: string(l.Data),
						Time: timestamppb.Now(),
						Type: pb.LogLine_TYPE_STDOUT,
					}:
						return nil
					}
				}); err != nil {
					slog.Error(
						"failed to tail log",
						slog.Uint64("procID", uint64(id)),
						slog.String("file", proc.StdoutFile),
						slog.Any("err", err),
					)
					// TODO: somehow call wg.Done() once with parent call
				}
			}()

			wgLocal.Add(1)
			stderrTailer := tail.File(proc.StdoutFile, tail.Config{
				Follow:        true,
				BufferSize:    _procLogsBufferSize,
				NotifyTimeout: time.Minute,
				Location:      &tail.Location{Whence: io.SeekEnd, Offset: -1000}, // TODO: limit offset by file size as with daemon logs
			})
			go func() {
				if err := stderrTailer.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
					select {
					case <-ctx.Done():
						wgGlobal.Done()
						return ctx.Err()
					case logLinesCh <- &pb.LogLine{
						Line: string(l.Data),
						Time: timestamppb.Now(),
						Type: pb.LogLine_TYPE_STDERR,
					}:
					}
					return nil
				}); err != nil {
					slog.Error(
						"failed to tail log",
						slog.Uint64("procID", uint64(id)),
						slog.String("file", proc.StdoutFile),
						slog.Any("err", err),
					)
					// TODO: somehow call wg.Done() once with parent call
				}
			}()

			go func() {
				wgLocal.Wait()
				close(logLinesCh)
			}()

			for {
				select {
				case <-done:
					return
				case line, ok := <-logLinesCh:
					if !ok {
						return
					}

					if errSend := stream.Send(&pb.ProcsLogs{
						Id:    uint64(id),
						Lines: []*pb.LogLine{line}, // TODO: collect lines for some time and send all at once
					}); errSend != nil {
						slog.Error(
							"failed to send log lines",
							slog.Uint64("procID", uint64(id)),
							slog.Any("err", errSend),
						)
						return
					}
				}
			}
		}(core.ProcID(id))
	}

	allStopped := make(chan struct{})
	go func() {
		wgGlobal.Wait()
		close(allStopped)
	}()

	go func() {
		defer close(done)

		select {
		case <-allStopped:
			return
		case <-stream.Context().Done():
			return
		}
	}()

	<-done
	return nil
}
