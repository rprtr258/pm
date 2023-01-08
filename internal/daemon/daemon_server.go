package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/samber/lo"
	"go.uber.org/multierr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/db"
)

var (
	_userHome      = os.Getenv("HOME")
	HomeDir        = path.Join(_userHome, ".pm")
	_daemonLogsDir = path.Join(HomeDir, "logs")
)

// TODO: logs for daemon everywhere
type daemonServer struct {
	api.UnimplementedDaemonServer
	db      db.DBHandle
	homeDir string
}

// Start - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (srv *daemonServer) Start(ctx context.Context, req *api.IDs) (*emptypb.Empty, error) {
	procs, err := srv.db.GetProcs(lo.Map(req.GetIds(), func(id *api.ProcessID, _ int) db.ProcID {
		return db.ProcID(id.GetId())
	}))
	if err != nil {
		return nil, fmt.Errorf("daemon.Start failed: %w", err)
	}

	for _, proc := range procs {
		procIDStr := strconv.FormatUint(uint64(proc.ID), 10)
		logsDir := path.Join(srv.homeDir, "logs")

		if err := os.Mkdir(logsDir, os.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("mkdir %s failed: %w", logsDir, err)
		}

		stdoutLogFilename := path.Join(logsDir, procIDStr+".stdout")
		stdoutLogFile, err := os.OpenFile(stdoutLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0660)
		if err != nil {
			return nil, fmt.Errorf("os.OpenFile(%s) failed: %w", stdoutLogFilename, err)
		}
		defer stdoutLogFile.Close()

		stderrLogFilename := path.Join(logsDir, procIDStr+".stderr")
		stderrLogFile, err := os.OpenFile(stderrLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0660)
		if err != nil {
			return nil, fmt.Errorf("os.OpenFile(%s) failed: %w", stderrLogFilename, err)
		}
		defer stderrLogFile.Close() // TODO: wrap
		// TODO: syscall.CloseOnExec(pidFile.Fd()) or just close pid file

		cwd, err := os.Getwd() // TODO: ???
		if err != nil {
			return nil, fmt.Errorf("os.Getwd failed: %w", err)
		}

		procAttr := os.ProcAttr{
			Dir: cwd,
			Env: os.Environ(), // TODO: ???
			Files: []*os.File{ // TODO: pid file is somehow passes to child
				os.Stdin,
				stdoutLogFile,
				stderrLogFile,
				// TODO: very fucking dirty hack not to inherit pid (and possibly other fds from daemon)
				// because I tried different variants, none of them worked out, including setting O_CLOEXEC on
				// pid file open and fcntl FD_CLOEXEC on already opened pid file fd
				nil, nil, nil, nil, nil, nil, nil, nil,
			},
			Sys: &syscall.SysProcAttr{
				Setpgid: true,
			},
		}

		args := append([]string{proc.Command}, proc.Args...)
		process, err := os.StartProcess(proc.Command, args, &procAttr)
		if err != nil {
			if err2 := srv.db.SetStatus(proc.ID, db.Status{Status: db.StatusErrored}); err2 != nil {
				return nil, fmt.Errorf("running failed, setting errored status failed: %w", multierr.Combine(err, err2))
			}

			return nil, fmt.Errorf("running failed process=%s: %w", spew.Sdump(proc), err)
		}

		runningStatus := db.Status{
			Status:    db.StatusRunning,
			Pid:       process.Pid,
			StartTime: time.Now(),
		}
		if err := srv.db.SetStatus(proc.ID, runningStatus); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

// Stop - stop processes by their ids in database
// TODO: change to sending signals
func (srv *daemonServer) Stop(_ context.Context, req *api.IDs) (*emptypb.Empty, error) {
	procsToStop := lo.Map(req.GetIds(), func(id *api.ProcessID, _ int) db.ProcID {
		return db.ProcID(id.GetId())
	})

	procsWeHaveAmongRequested, err := srv.db.GetProcs(procsToStop)
	if err != nil {
		return nil, fmt.Errorf("getting procs to stop from db failed: %w", err)
	}

	var merr error
	for _, proc := range procsWeHaveAmongRequested {
		multierr.AppendInto(&merr, srv.stop(proc))
	}

	return &emptypb.Empty{}, merr
}

func (srv *daemonServer) stop(proc db.ProcData) error {
	if proc.Status.Status != db.StatusRunning {
		// TODO: structured logging, INFO here
		log.Printf("proc %+v was asked to be stopped, but not running\n", proc)
		return nil
	}

	// TODO: actually stop proc
	process, err := os.FindProcess(proc.Status.Pid)
	if err != nil {
		return fmt.Errorf("getting process by pid=%d failed: %w", proc.Status.Pid, err)
	}

	// TODO: kill after timeout
	if err := syscall.Kill(-process.Pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			log.Printf("[WARN] finished process %+v with running status", proc)
		} else {
			return fmt.Errorf("killing process with pid=%d failed: %w", process.Pid, err)
		}
	}

	state, err := process.Wait()
	var errno syscall.Errno
	if err != nil {
		if errors.As(err, &errno); errno != 10 {
			return fmt.Errorf("releasing process %+v failed: %w %#v", proc, err, spew.Sdump(err))
		} else {
			fmt.Printf("[INFO] process %+v is not a child", proc)
		}
	} else {
		fmt.Printf("[INFO] process %+v closed with state %+v\n", proc, state)
	}

	if err := srv.db.SetStatus(proc.ID, db.Status{Status: db.StatusStopped}); err != nil {
		return status.Errorf(codes.DataLoss, fmt.Errorf("updating status of process %+v failed: %w", proc, err).Error())
	}

	return nil
}

func (srv *daemonServer) Create(ctx context.Context, r *api.ProcessOptions) (*api.ProcessID, error) {
	if r.Name != nil {
		procs, err := srv.db.List()
		if err != nil {
			return nil, err
		}

		if procID, ok := lo.FindKeyBy(
			procs,
			func(_ db.ProcID, procData db.ProcData) bool {
				return procData.Name == r.GetName()
			},
		); ok {
			return &api.ProcessID{Id: uint64(procID)}, nil
		}
	}

	name := lo.IfF(r.Name != nil, r.GetName).ElseF(genName)

	procData := db.ProcData{
		Status: db.Status{
			Status: db.StatusStarting,
		},
		Name:    name,
		Cwd:     ".",
		Tags:    lo.Uniq(append(r.GetTags(), "all")),
		Command: r.GetCommand(),
		Args:    r.GetArgs(),
	}

	procID, err := srv.db.AddProc(procData)
	if err != nil {
		return nil, err
	}

	return &api.ProcessID{Id: uint64(procID)}, nil
}

func genName() string {
	return petname.Generate(2, "-")
}

func (srv *daemonServer) List(ctx context.Context, _ *emptypb.Empty) (*api.ProcessesList, error) {
	list, err := srv.db.List()
	if err != nil {
		return nil, err
	}

	return &api.ProcessesList{
		List: lo.MapToSlice(
			list,
			func(id db.ProcID, proc db.ProcData) *api.Process {
				return &api.Process{
					Id:      &api.ProcessID{Id: uint64(id)},
					Status:  mapStatus(proc.Status),
					Name:    proc.Name,
					Cwd:     proc.Cwd,
					Tags:    proc.Tags,
					Command: proc.Command,
					Args:    proc.Args,
				}
			},
		),
	}, nil
}

func mapStatus(status db.Status) *api.ProcessStatus {
	switch status.Status {
	case db.StatusInvalid:
		return &api.ProcessStatus{Status: &api.ProcessStatus_Invalid{}}
	case db.StatusErrored:
		// TODO: exit status code
		return &api.ProcessStatus{Status: &api.ProcessStatus_Errored{}}
	case db.StatusStarting:
		return &api.ProcessStatus{Status: &api.ProcessStatus_Starting{}}
	case db.StatusStopped:
		return &api.ProcessStatus{Status: &api.ProcessStatus_Stopped{}}
	case db.StatusRunning:
		return &api.ProcessStatus{Status: &api.ProcessStatus_Running{
			Running: &api.RunningProcessStatus{
				Pid:       int64(status.Pid),
				StartTime: timestamppb.New(status.StartTime),
				Cpu:       status.Cpu,
				Memory:    status.Memory,
			},
		}}
	default:
		return &api.ProcessStatus{Status: &api.ProcessStatus_Invalid{}}
	}
}

func (srv *daemonServer) Delete(ctx context.Context, r *api.IDs) (*emptypb.Empty, error) {
	ids := lo.Map(
		r.GetIds(),
		func(procID *api.ProcessID, _ int) uint64 {
			return procID.GetId()
		},
	)
	if err := srv.db.Delete(ids); err != nil {
		return nil, err // TODO: add errs descriptions, loggings
	}

	var merr error
	for _, procID := range ids {
		if err := removeLogFiles(procID); err != nil {
			multierr.AppendInto(&merr, fmt.Errorf("couldn't delete proc #%d: %w", procID, err))
		}
	}

	return &emptypb.Empty{}, merr
}

func removeLogFiles(procID uint64) error {
	stdoutFilename := filepath.Join(_daemonLogsDir, fmt.Sprintf("%d.stdout", procID))
	if err := removeFile(stdoutFilename); err != nil {
		return err
	}

	stderrFilename := filepath.Join(_daemonLogsDir, fmt.Sprintf("%d.stderr", procID))
	if err := removeFile(stderrFilename); err != nil {
		return err
	}

	return nil
}

func removeFile(name string) error {
	_, err := os.Stat(name)
	if err == os.ErrNotExist {
		return nil
	} else if err != nil {
		return err
	}

	return os.Remove(name)
}

func (srv *daemonServer) HealthCheck(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
