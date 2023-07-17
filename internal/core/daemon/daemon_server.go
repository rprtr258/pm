package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-faster/tail"
	fun2 "github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/fun"
	"github.com/rprtr258/pm/internal/core/namegen"
	"github.com/rprtr258/pm/internal/infra/db"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	db               db.Handle
	homeDir, logsDir string
}

func (srv *daemonServer) start(proc db.ProcData) error {
	if procs := srv.db.GetProcs([]core.ProcID{proc.ProcID}); len(procs) > 0 {
		if len(procs) > 1 {
			return xerr.NewF("invalid procs count got by id", xerr.Fields{
				"id":    proc.ProcID,
				"procs": procs,
			})
		}

		if procs[0].Status.Status == db.StatusRunning {
			return xerr.NewF("process is already running", xerr.Fields{"id": proc.ProcID})
		}
	}

	procIDStr := strconv.FormatUint(uint64(proc.ProcID), 10) //nolint:gomnd // decimal
	stdoutLogFilename := path.Join(srv.logsDir, procIDStr+".stdout")
	stdoutLogFile, err := os.OpenFile(stdoutLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return xerr.NewWM(err, "open stdout file", xerr.Fields{"filename": stdoutLogFile})
	}
	defer stdoutLogFile.Close()

	stderrLogFilename := path.Join(srv.logsDir, procIDStr+".stderr")
	stderrLogFile, err := os.OpenFile(stderrLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return xerr.NewWM(err, "open stderr file", xerr.Fields{"filename": stderrLogFilename})
	}
	defer func() {
		if errClose := stderrLogFile.Close(); errClose != nil {
			slog.Error(errClose.Error())
		}
	}()
	// TODO: syscall.CloseOnExec(pidFile.Fd()) or just close pid file

	env := os.Environ()
	for k, v := range proc.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	procAttr := os.ProcAttr{
		Dir: proc.Cwd,
		Env: env,
		Files: []*os.File{ // TODO: pid file is somehow passes to child
			0: os.Stdin,
			1: stdoutLogFile,
			2: stderrLogFile,
			// TODO: very fucking dirty hack not to inherit pid (and possibly other fds from daemon)
			// because I tried different variants, none of them worked out, including setting O_CLOEXEC on
			// pid file open and fcntl FD_CLOEXEC on already opened pid file fd
			// TODO: try if syscall.Getuid() == 0 {syscall.Setgid(GID) == nil && syscall.Setuid(UID) == nil}
			/* 3,4,5,6,7... */ nil, nil, nil, nil, nil, nil, nil, nil,
		},
		Sys: &syscall.SysProcAttr{
			Setpgid: true,
		},
	}

	args := append([]string{proc.Command}, proc.Args...)
	process, err := os.StartProcess(proc.Command, args, &procAttr)
	if err != nil {
		if errSetStatus := srv.db.SetStatus(proc.ProcID, db.NewStatusInvalid()); errSetStatus != nil {
			return xerr.NewWM(xerr.Combine(err, errSetStatus), "running failed, setting errored status failed")
		}

		return xerr.NewWM(err, "running failed", xerr.Fields{"procData": spew.Sprint(proc)})
	}

	runningStatus := db.NewStatusRunning(time.Now(), process.Pid)
	if err := srv.db.SetStatus(proc.ProcID, runningStatus); err != nil {
		return xerr.NewWM(err, "set status running", xerr.Fields{"procID": proc.ProcID})
	}

	return nil
}

// Start - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (srv *daemonServer) Start(ctx context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	procs := srv.db.GetProcs(lo.Map(req.GetIds(), func(id *pb.ProcessID, _ int) core.ProcID {
		return core.ProcID(id.GetId())
	}))

	for _, proc := range procs {
		select {
		case <-ctx.Done():
			return nil, xerr.NewWM(ctx.Err(), "context canceled")
		default:
		}

		if errStart := srv.start(proc); errStart != nil {
			return nil, xerr.NewW(errStart, xerr.Fields{"proc": proc})
		}
	}

	return &emptypb.Empty{}, nil
}

// Signal - send signal processes to processes
func (srv *daemonServer) Signal(_ context.Context, req *pb.SignalRequest) (*emptypb.Empty, error) {
	procsToStop := lo.Map(req.GetIds(), func(id *pb.ProcessID, _ int) core.ProcID {
		return core.ProcID(id.GetId())
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

func (srv *daemonServer) stop(ctx context.Context, proc db.ProcData) (bool, error) {
	if proc.Status.Status != db.StatusRunning {
		slog.Info("tried to stop non-running process", "proc", proc)
		return false, nil
	}

	process, errFindProc := os.FindProcess(proc.Status.Pid)
	if errFindProc != nil {
		return false, xerr.NewWM(errFindProc, "find process", xerr.Fields{"pid": proc.Status.Pid})
	}

	if errKill := syscall.Kill(-process.Pid, syscall.SIGTERM); errKill != nil {
		switch {
		case errors.Is(errKill, os.ErrProcessDone):
			slog.Warn("tried stop process which is done", "proc", proc)
		case errors.Is(errKill, syscall.ESRCH): // no such process
			slog.Warn("tried stop process which doesn't exist", "proc", proc)
		default:
			return false, xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": process.Pid})
		}
	}

	doneCh := make(chan *os.ProcessState, 1)
	go func() {
		state, errFindProc := process.Wait()
		if errFindProc != nil {
			if errno, ok := xerr.As[syscall.Errno](errFindProc); !ok || errno != 10 {
				slog.Error("releasing process",
					"pid", process.Pid,
					"err", errFindProc.Error(),
				)
				doneCh <- nil
				return
			}

			slog.Info("process is not a child", "proc", proc)
		} else {
			slog.Info("process is stopped", "proc", proc, "state", state)
		}
		doneCh <- state
	}()

	timer := time.NewTimer(time.Second * 5) //nolint:gomnd // arbitrary timeout
	defer timer.Stop()

	var state *os.ProcessState
	select {
	case <-timer.C:
		slog.Warn("timed out waiting for process to stop from SIGTERM, killing it", "proc", proc)

		if errKill := syscall.Kill(-process.Pid, syscall.SIGKILL); errKill != nil {
			return false, xerr.NewWM(errKill, "kill process", xerr.Fields{"pid": process.Pid})
		}
	case <-ctx.Done():
		return false, xerr.NewWM(ctx.Err(), "context canceled")
	case state = <-doneCh:
	}

	exitCode := lo.If(state == nil, -1).ElseF(func() int {
		return state.ExitCode()
	})
	if errSetStatus := srv.db.SetStatus(proc.ProcID, db.NewStatusStopped(exitCode)); errSetStatus != nil {
		return false, xerr.NewWM(errSetStatus, "set status stopped", xerr.Fields{"procID": proc.ProcID})
	}

	return true, nil
}

func (srv *daemonServer) Stop(ctx context.Context, req *pb.IDs) (*pb.IDs, error) {
	procsList := srv.db.GetProcs(lo.Map(req.GetIds(), func(id *pb.ProcessID, _ int) core.ProcID {
		return core.ProcID(id.GetId())
	}))

	procs := lo.SliceToMap(procsList, func(proc db.ProcData) (core.ProcID, db.ProcData) {
		return proc.ProcID, proc
	})

	stoppedIDs := []core.ProcID{}

	var merr error
	for _, procID := range req.GetIds() {
		select {
		case <-ctx.Done():
			return nil, xerr.NewWM(ctx.Err(), "context canceled")
		default:
		}

		proc, ok := procs[core.ProcID(procID.GetId())]
		if !ok {
			slog.Info("tried to stop non-existing process", "proc", procID.GetId())
			continue
		}

		stopped, errStop := srv.stop(ctx, proc)
		if errStop != nil {
			xerr.AppendInto(&merr, errStop)
			continue
		}

		if stopped {
			stoppedIDs = append(stoppedIDs, proc.ProcID)
		}
	}

	return &pb.IDs{
		Ids: lo.Map(stoppedIDs, func(id core.ProcID, _ int) *pb.ProcessID {
			return &pb.ProcessID{Id: uint64(id)}
		}),
	}, nil
}

func (srv *daemonServer) signal(proc db.ProcData, signal syscall.Signal) error {
	if proc.Status.Status != db.StatusRunning {
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
	procIDs := make([]core.ProcID, len(req.GetOptions()))
	for i, opts := range req.GetOptions() {
		var errCreate error
		procIDs[i], errCreate = srv.create(ctx, opts)
		if errCreate != nil {
			return nil, errCreate
		}
	}

	return &pb.IDs{
		Ids: lo.Map(procIDs, func(procID core.ProcID, _ int) *pb.ProcessID {
			return &pb.ProcessID{
				Id: uint64(procID),
			}
		}),
	}, nil
}

func (srv *daemonServer) create(ctx context.Context, procOpts *pb.ProcessOptions) (core.ProcID, error) {
	if procOpts.Name != nil {
		procs := srv.db.List()

		if procID, ok := lo.FindKeyBy(
			procs,
			func(_ core.ProcID, procData db.ProcData) bool {
				return procData.Name == procOpts.GetName()
			},
		); ok {
			procData := db.ProcData{
				ProcID:  procID,
				Status:  db.NewStatusCreated(),
				Name:    procOpts.GetName(),
				Cwd:     procOpts.GetCwd(),
				Tags:    lo.Uniq(append(procOpts.GetTags(), "all")),
				Command: procOpts.GetCommand(),
				Args:    procOpts.GetArgs(),
				Watch:   nil,
				Env:     procOpts.GetEnv(),
			}

			proc := procs[procID]
			if proc.Status.Status != db.StatusRunning ||
				proc.Cwd == procData.Cwd &&
					len(proc.Tags) == len(procData.Tags) && // TODO: compare lists, not lengths
					proc.Command == procData.Command &&
					len(proc.Args) == len(procData.Args) && // TODO: compare lists, not lengths
					(proc.Watch == nil) == (procData.Watch == nil) && (proc.Watch == nil || *proc.Watch == *procData.Watch) { // TODO: compare pointers
				// not updated, do nothing
				return procID, nil
			}

			if _, errStop := srv.stop(ctx, proc); errStop != nil {
				return 0, xerr.NewWM(errStop, "stop process to update", xerr.Fields{
					"procID":  procID,
					"oldProc": proc,
					"newProc": procData,
				})
			}

			if errUpdate := srv.db.UpdateProc(procData); errUpdate != nil {
				return 0, xerr.NewWM(errUpdate, "update proc", xerr.Fields{"procData": procData})
			}

			return procID, nil
		}
	}

	name := fun.IfF(procOpts.Name != nil, procOpts.GetName).ElseF(namegen.New)

	procID, err := srv.db.AddProc(db.CreateQuery{
		Name:    name,
		Cwd:     procOpts.GetCwd(),
		Tags:    lo.Uniq(append(procOpts.GetTags(), "all")),
		Command: procOpts.GetCommand(),
		Args:    procOpts.GetArgs(),
		Watch:   fun2.Option[string]{},
		Env:     procOpts.Env,
	})
	if err != nil {
		return 0, xerr.NewWM(err, "save proc")
	}

	return procID, nil
}

func (srv *daemonServer) List(ctx context.Context, _ *emptypb.Empty) (*pb.ProcessesList, error) {
	// TODO: update statuses here also
	list := srv.db.List()

	return &pb.ProcessesList{
		Processes: lo.MapToSlice(list, func(id core.ProcID, proc db.ProcData) *pb.Process {
			return &pb.Process{
				Id:      &pb.ProcessID{Id: uint64(id)},
				Status:  mapStatus(proc.Status),
				Name:    proc.Name,
				Cwd:     proc.Cwd,
				Tags:    proc.Tags,
				Command: proc.Command,
				Args:    proc.Args,
			}
		}),
	}, nil
}

//nolint:exhaustruct // can't return api.isProcessStatus_Status
func mapStatus(status db.Status) *pb.ProcessStatus {
	switch status.Status {
	case db.StatusInvalid:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Invalid{}}
	case db.StatusCreated:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Created{}}
	case db.StatusStopped:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Stopped{
			Stopped: &pb.StoppedProcessStatus{
				ExitCode:  int64(status.ExitCode),
				StoppedAt: timestamppb.New(status.StoppedAt),
			},
		}}
	case db.StatusRunning:
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
	ids := lo.Map(
		r.GetIds(),
		func(procID *pb.ProcessID, _ int) uint64 {
			return procID.GetId()
		},
	)
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

func (srv *daemonServer) Logs(req *pb.IDs, stream pb.Daemon_LogsServer) error {
	// can't get incoming query in interceptor, so logging here also
	slog.Info("Logs method called",
		slog.Any("ids", req.GetIds()),
	)

	procs := srv.db.List() // TODO: filter by ids
	done := make(chan struct{})

	var wg sync.WaitGroup
	for _, id := range req.GetIds() {
		proc, ok := procs[core.ProcID(id.GetId())]
		if !ok {
			slog.Info("tried to log unknown process", "procID", id.GetId())
			continue
		}
		if proc.Status.Status != db.StatusRunning {
			slog.Info("tried to log non-running process", "procID", id.GetId())
			continue
		}

		wg.Add(2)
		go func(id core.ProcID) {
			defer wg.Done()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// TODO: proc.StdoutFile, proc.StderrFile
			stdoutTailer := tail.File(filepath.Join(srv.logsDir, fmt.Sprintf("%d.stdout", id)), tail.Config{
				Follow:        true,
				BufferSize:    1024 * 128, // 128 kb
				NotifyTimeout: time.Minute,
				Location:      &tail.Location{Whence: io.SeekEnd, Offset: -1000}, // TODO: limit offset by file size as with daemon logs
			})
			stdoutLinesCh := make(chan []byte)
			go func() {
				if err := stdoutTailer.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
					select {
					case <-ctx.Done():
						close(stdoutLinesCh)
					case stdoutLinesCh <- append([]byte(nil), l.Data...):
					}
					return nil
				}); err != nil {
					slog.Error("failed to tail log", slog.Uint64("procID", uint64(id)), slog.Any("err", err))
					// TODO: somehow call wg.Done() once with parent call
				}
			}()

			stderrTailer := tail.File(filepath.Join(srv.logsDir, fmt.Sprintf("%d.stderr", id)), tail.Config{
				Follow:        true,
				BufferSize:    1024 * 128, // 128 kb
				NotifyTimeout: time.Minute,
				Location:      &tail.Location{Whence: io.SeekEnd, Offset: -1000}, // TODO: limit offset by file size as with daemon logs
			})
			stderrLinesCh := make(chan []byte)
			go func() {
				if err := stderrTailer.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
					select {
					case <-ctx.Done():
						close(stderrLinesCh)
					case stderrLinesCh <- append([]byte(nil), l.Data...):
					}
					return nil
				}); err != nil {
					slog.Error("failed to tail log", slog.Uint64("procID", uint64(id)), slog.Any("err", err))
					// TODO: somehow call wg.Done() once with parent call
				}
			}()

			var (
				line     string
				lineType pb.LogLine_Type
			)
			for {
				select {
				case <-done:
					return
				case lineBytes := <-stdoutLinesCh:
					line, lineType = string(lineBytes), pb.LogLine_TYPE_STDOUT
				case lineBytes := <-stderrLinesCh:
					line, lineType = string(lineBytes), pb.LogLine_TYPE_STDERR
				}

				if errSend := stream.Send(&pb.ProcsLogs{
					Id: uint64(id),
					Lines: []*pb.LogLine{ // TODO: collect lines for some time and send all at once
						{
							Line: line,
							Time: timestamppb.Now(),
							Type: lineType,
						},
					},
				}); errSend != nil {
					slog.Error(
						"failed to send log lines",
						slog.Uint64("procID", uint64(id)),
						slog.Any("err", errSend),
					)
					return
				}
			}
		}(core.ProcID(id.GetId()))
	}

	allStopped := make(chan struct{})
	go func() {
		wg.Wait()
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
