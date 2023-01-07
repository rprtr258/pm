package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/db"
	"github.com/samber/lo"
	"go.uber.org/multierr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO: logs for daemon everywhere
type daemonServer struct {
	pb.UnimplementedDaemonServer
	dbFile  string
	homeDir string
}

// Start - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (srv *daemonServer) Start(ctx context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	procs, err := db.New(srv.dbFile).GetProcs(lo.Map(req.GetIds(), func(id *pb.ProcessID, _ int) db.ProcID {
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

		process, err := os.StartProcess(proc.Command, append([]string{proc.Command}, proc.Args...), &procAttr)
		if err != nil {
			if err2 := db.New(srv.dbFile).SetStatus(proc.ID, db.Status{Status: db.StatusErrored}); err2 != nil {
				return nil, fmt.Errorf("running failed, setting errored status failed: %w", multierr.Combine(err, err2))
			}

			return nil, fmt.Errorf("running failed process=%s: %w", spew.Sdump(proc), err)
		}

		runningStatus := db.Status{
			Status:    db.StatusRunning,
			Pid:       process.Pid,
			StartTime: time.Now(),
		}
		if err := db.New(srv.dbFile).SetStatus(proc.ID, runningStatus); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

// Stop - stop processes by their ids in database
// TODO: change to sending signals
func (srv *daemonServer) Stop(_ context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	dbHandle := db.New(srv.dbFile)

	procsToStop := lo.Map(req.GetIds(), func(id *pb.ProcessID, _ int) db.ProcID {
		return db.ProcID(id.GetId())
	})

	procsWeHaveAmongRequested, err := dbHandle.GetProcs(procsToStop)
	if err != nil {
		return nil, fmt.Errorf("getting procs to stop from db failed: %w", err)
	}

	for _, proc := range procsWeHaveAmongRequested {
		if proc.Status.Status != db.StatusRunning {
			// TODO: structured logging, INFO here
			log.Printf("proc %+v was asked to be stopped, but not running\n", proc)
			continue
		}

		// TODO: actually stop proc
		process, err := os.FindProcess(proc.Status.Pid)
		if err != nil {
			return nil, fmt.Errorf("getting process by pid=%d failed: %w", proc.Status.Pid, err)
		}

		// TODO: kill after timeout
		if err := syscall.Kill(-process.Pid, syscall.SIGTERM); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				log.Printf("[WARN] finished process %+v with running status", proc)
			} else {
				return nil, fmt.Errorf("killing process with pid=%d failed: %w", process.Pid, err)
			}
		}

		state, err := process.Wait()
		var errno syscall.Errno
		if err != nil {
			if errors.As(err, &errno); errno != 10 {
				return nil, fmt.Errorf("releasing process %+v failed: %w %#v", proc, err, spew.Sdump(err))
			} else {
				fmt.Printf("[INFO] process %+v is not a child", proc)
			}
		} else {
			fmt.Printf("[INFO] process %+v closed with state %+v\n", proc, state)
		}

		if err := dbHandle.SetStatus(proc.ID, db.Status{Status: db.StatusStopped}); err != nil {
			return nil, status.Errorf(codes.DataLoss, fmt.Errorf("updating status of process %+v failed: %w", proc, err).Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (srv *daemonServer) Create(ctx context.Context, r *pb.ProcessOptions) (*pb.ProcessID, error) {
	procData := db.ProcData{
		Status: db.Status{
			Status: db.StatusStarting,
		},
		Name:    *r.Name,
		Cwd:     ".",
		Tags:    lo.Uniq(append(r.GetTags(), "all")),
		Command: r.GetCommand(),
		Args:    r.GetArgs(),
	}

	procID, err := db.New(srv.dbFile).AddProc(procData)
	if err != nil {
		return nil, err
	}

	return &pb.ProcessID{Id: uint64(procID)}, nil
}

func (srv *daemonServer) List(ctx context.Context, _ *emptypb.Empty) (*pb.ProcessesList, error) {
	dbHandle := db.New(srv.dbFile)

	list, err := dbHandle.List()
	if err != nil {
		return nil, err
	}

	return &pb.ProcessesList{
		List: lo.MapToSlice(
			list,
			func(id db.ProcID, proc db.ProcData) *pb.Process {
				return &pb.Process{
					Id:      &pb.ProcessID{Id: uint64(id)},
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

func mapStatus(status db.Status) *pb.ProcessStatus {
	switch status.Status {
	case db.StatusInvalid:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Invalid{}}
	case db.StatusErrored:
		// TODO: exit status code
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Errored{}}
	case db.StatusStarting:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Starting{}}
	case db.StatusStopped:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Stopped{}}
	case db.StatusRunning:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Running{
			Running: &pb.RunningProcessStatus{
				Pid:       int64(status.Pid),
				StartTime: timestamppb.New(status.StartTime),
				Cpu:       status.Cpu,
				Memory:    status.Memory,
			},
		}}
	default:
		return &pb.ProcessStatus{Status: &pb.ProcessStatus_Invalid{}}
	}
}
