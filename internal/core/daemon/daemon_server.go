package daemon

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
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
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/runner"
	"github.com/rprtr258/pm/internal/infra/db"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	db               db.Handle
	ebus             *eventbus.EventBus
	homeDir, logsDir string
	runner           runner.Runner // TODO: remove, used only for create
}

// Start - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (srv *daemonServer) Start(ctx context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	for _, id := range req.GetIds() {
		srv.ebus.PublishProcStartRequest(id, eventbus.EmitReasonByUser)
	}

	return &emptypb.Empty{}, nil
}

// Signal - send signal processes to processes
func (srv *daemonServer) Signal(ctx context.Context, req *pb.SignalRequest) (*emptypb.Empty, error) {
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

	srv.ebus.PublishProcSignalRequest(signal, req.GetIds()...)

	return &emptypb.Empty{}, nil
}

func (srv *daemonServer) Stop(ctx context.Context, req *pb.IDs) (*pb.IDs, error) {
	for _, id := range req.GetIds() {
		srv.ebus.PublishProcStopRequest(id, eventbus.EmitReasonByUser)
	}

	return &pb.IDs{
		// Ids: stoppedIDs, // TODO: implement somehow?
	}, nil
}

func (srv *daemonServer) Create(ctx context.Context, req *pb.CreateRequest) (*pb.IDs, error) {
	queries := lo.Map(
		req.GetOptions(),
		func(opts *pb.ProcessOptions, _ int) runner.CreateQuery {
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
		},
	)
	procIDs, err := srv.runner.Create(ctx, queries...)
	if err != nil {
		return nil, err
	}

	return &pb.IDs{
		Ids: procIDs,
	}, nil
}

func (srv *daemonServer) List(_ context.Context, _ *emptypb.Empty) (*pb.ProcessesList, error) {
	return &pb.ProcessesList{
		Processes: lo.MapToSlice(
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

func (srv *daemonServer) Delete(_ context.Context, r *pb.IDs) (*emptypb.Empty, error) {
	ids := r.GetIds()
	deletedProcs, errDelete := srv.db.Delete(ids)
	if errDelete != nil {
		return nil, xerr.NewWM(errDelete, "delete proc", xerr.Fields{"procIDs": ids})
	}

	var merr error
	for _, proc := range deletedProcs {
		if err := removeLogFiles(proc); err != nil {
			xerr.AppendInto(&merr, xerr.NewWM(err, "delete proc", xerr.Fields{"procID": proc.ID}))
		}
	}

	return &emptypb.Empty{}, merr
}

func removeLogFiles(proc core.Proc) error {
	if errRmStdout := removeFile(proc.StdoutFile); errRmStdout != nil {
		return xerr.NewWM(errRmStdout, "remove stdout file", xerr.Fields{"stdout_file": proc.StdoutFile})
	}

	if errRmStderr := removeFile(proc.StderrFile); errRmStderr != nil {
		return xerr.NewWM(errRmStderr, "remove stderr file", xerr.Fields{"stderr_file": proc.StderrFile})
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

func (srv *daemonServer) HealthCheck(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

const (
	_procLogsBufferSize = 128 * 1024 // 128 kibibytes
	_defaultLogsOffset  = 1000       // 100 bytes
)

func streamFile(
	ctx context.Context,
	logLinesCh chan *pb.LogLine,
	procID core.ProcID,
	logFile string,
	logLineType pb.LogLine_Type,
	wgGlobal *sync.WaitGroup,
) error {
	stat, errStat := os.Stat(logFile)
	if errStat != nil {
		return xerr.NewWM(errStat, "stat log file")
	}

	tailer := tail.File(logFile, tail.Config{
		Follow:        true,
		BufferSize:    _procLogsBufferSize,
		NotifyTimeout: time.Minute,
		Location: &tail.Location{
			Whence: io.SeekEnd,
			Offset: -fun.Min(stat.Size(), _defaultLogsOffset),
		},
		Logger:  nil,
		Tracker: nil,
	})
	go func() {
		if err := tailer.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
			select {
			case <-ctx.Done():
				wgGlobal.Done()
				return ctx.Err()
			case logLinesCh <- &pb.LogLine{
				Line: string(l.Data),
				Time: timestamppb.Now(),
				Type: logLineType,
			}:
				return nil
			}
		}); err != nil {
			slog.Error(
				"failed to tail log",
				slog.Uint64("procID", procID),
				slog.String("file", logFile),
				slog.Any("err", err),
			)
			// TODO: somehow call wg.Done() once with parent call
		}
	}()

	return nil
}

func (srv *daemonServer) Logs(req *pb.IDs, stream pb.Daemon_LogsServer) error {
	// can't get incoming query in interceptor, so logging here also
	slog.Info("Logs method called",
		slog.Any("ids", req.GetIds()),
	)

	procs := srv.db.GetProcs(core.WithIDs(req.GetIds()...))
	done := make(chan struct{})

	var wgGlobal sync.WaitGroup
	for _, id := range req.GetIds() {
		proc, ok := procs[id]
		if !ok {
			slog.Info("tried to log unknown process", slog.Uint64("procID", id))
			continue
		}

		wgGlobal.Add(2)
		go func(id core.ProcID) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var wgLocal sync.WaitGroup

			logLinesCh := make(chan *pb.LogLine)

			wgLocal.Add(1)
			if err := streamFile(
				ctx,
				logLinesCh,
				proc.ID,
				proc.StdoutFile,
				pb.LogLine_TYPE_STDOUT,
				&wgGlobal,
			); err != nil {
				slog.Error(
					"failed to stream stdout log file",
					slog.Uint64("procID", id),
					slog.String("file", proc.StdoutFile),
					slog.Any("err", err),
				)
			}

			wgLocal.Add(1)
			if err := streamFile(
				ctx,
				logLinesCh,
				proc.ID,
				proc.StderrFile,
				pb.LogLine_TYPE_STDERR,
				&wgGlobal,
			); err != nil {
				slog.Error(
					"failed to stream stderr log file",
					slog.Uint64("procID", id),
					slog.String("file", proc.StderrFile),
					slog.Any("err", err),
				)
			}

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
						Id:    id,
						Lines: []*pb.LogLine{line}, // TODO: collect lines for some time and send all at once
					}); errSend != nil {
						slog.Error(
							"failed to send log lines",
							slog.Uint64("procID", id),
							slog.Any("err", errSend),
						)
						return
					}
				}
			}
		}(id)
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
