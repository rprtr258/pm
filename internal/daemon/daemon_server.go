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

	"github.com/rprtr258/xerr"

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
		return nil, xerr.NewWM(err, "daemon.start")
	}

	for _, proc := range procs {
		procIDStr := strconv.FormatUint(uint64(proc.ProcID), 10)
		logsDir := path.Join(srv.homeDir, "logs")

		if err := os.Mkdir(logsDir, os.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, xerr.NewWM(err, "create logs dir", xerr.Field("dir", logsDir))
		}

		stdoutLogFilename := path.Join(logsDir, procIDStr+".stdout")
		stdoutLogFile, err := os.OpenFile(stdoutLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
		if err != nil {
			return nil, xerr.NewWM(err, "open stdout file", xerr.Field("filename", stdoutLogFile))
		}
		defer stdoutLogFile.Close()

		stderrLogFilename := path.Join(logsDir, procIDStr+".stderr")
		stderrLogFile, err := os.OpenFile(stderrLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
		if err != nil {
			return nil, xerr.NewWM(err, "open stderr file", xerr.Field("filename", stderrLogFilename))
		}
		defer stderrLogFile.Close() // TODO: wrap
		// TODO: syscall.CloseOnExec(pidFile.Fd()) or just close pid file

		cwd, err := os.Getwd()
		if err != nil {
			return nil, xerr.NewWM(err, "os.Getwd")
		}

		procAttr := os.ProcAttr{
			Dir: filepath.Join(cwd, proc.Cwd),
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
			if found := srv.db.SetStatus(proc.ProcID, db.Status{Status: db.StatusErrored}); !found {
				return nil, xerr.NewM("running failed, setting errored status failed",
					xerr.Errors(
						err,
						xerr.NewM("proc was not found", xerr.Field("procID", proc.ProcID))))
			}

			return nil, xerr.NewWM(err, "running failed", xerr.Field("procData", spew.Sdump(proc)))
		}

		runningStatus := db.Status{
			Status:    db.StatusRunning,
			Pid:       process.Pid,
			StartTime: time.Now(),
		}
		if found := srv.db.SetStatus(proc.ProcID, runningStatus); !found {
			return nil, xerr.NewM("proc was not found", xerr.Field("procID", proc.ProcID))
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
		return nil, xerr.NewWM(err, "getting procs to stop")
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
		return xerr.NewWM(err, "getting process by pid failed", xerr.Field("pid", proc.Status.Pid))
	}

	// TODO: kill after timeout
	if err := syscall.Kill(-process.Pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			log.Printf("[WARN] finished process %+v with running status", proc)
		} else {
			return xerr.NewWM(err, "killing process failed", xerr.Field("pid", process.Pid))
		}
	}

	state, err := process.Wait()
	if err != nil {
		if errno, ok := xerr.As[syscall.Errno](err); !ok || errno != 10 {
			return xerr.NewWM(err, "releasing process", xerr.Field("procData", proc))
		}

		fmt.Printf("[INFO] process %+v is not a child", proc)
	} else {
		fmt.Printf("[INFO] process %+v closed with state %+v\n", proc, state)
	}

	if found := srv.db.SetStatus(proc.ProcID, db.Status{Status: db.StatusStopped}); !found {
		return status.Errorf(codes.DataLoss, "updating process status, not found, pid=%d", proc.ProcID)
	}

	return nil
}

func (srv *daemonServer) Create(ctx context.Context, procOpts *api.ProcessOptions) (*api.ProcessID, error) {
	if procOpts.Name != nil {
		procs := srv.db.List()

		if procID, ok := lo.FindKeyBy(
			procs,
			func(_ db.ProcID, procData db.ProcData) bool {
				return procData.Name == procOpts.GetName()
			},
		); ok {
			procData := db.ProcData{
				ProcID: procID,
				Status: db.Status{
					Status: db.StatusStarting,
				},
				Name:    procOpts.GetName(),
				Cwd:     procOpts.GetCwd(),
				Tags:    lo.Uniq(append(procOpts.GetTags(), "all")),
				Command: procOpts.GetCommand(),
				Args:    procOpts.GetArgs(),
				Watch:   nil,
			}

			srv.db.UpdateProc(procData)

			return &api.ProcessID{Id: uint64(procID)}, nil
		}
	}

	name := lo.IfF(procOpts.Name != nil, procOpts.GetName).ElseF(genName)

	procData := db.ProcData{
		Status: db.Status{
			Status: db.StatusStarting,
		},
		Name:    name,
		Cwd:     procOpts.GetCwd(),
		Tags:    lo.Uniq(append(procOpts.GetTags(), "all")),
		Command: procOpts.GetCommand(),
		Args:    procOpts.GetArgs(),
		Watch:   nil,
	}

	procID, err := srv.db.AddProc(procData)
	if err != nil {
		return nil, xerr.NewWM(err, "save proc")
	}

	return &api.ProcessID{Id: uint64(procID)}, nil
}

func genName() string {
	const words = 2
	return petname.Generate(words, "-")
}

func (srv *daemonServer) List(ctx context.Context, _ *emptypb.Empty) (*api.ProcessesList, error) {
	list := srv.db.List()

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
				Cpu:       status.CPU,
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
	srv.db.Delete(ids)
	// TODO: add errs descriptions, loggings

	var merr error
	for _, procID := range ids {
		if err := removeLogFiles(procID); err != nil {
			multierr.AppendInto(&merr, xerr.NewWM(err, "delete proc", xerr.Field("procID", procID)))
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
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return xerr.NewWM(err, "remove file, stat",
			xerr.Field("filename", name))
	}

	if err := os.Remove(name); err != nil {
		return xerr.NewWM(err, "remove file",
			xerr.Field("filename", name))
	}

	return nil
}

func (srv *daemonServer) HealthCheck(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
