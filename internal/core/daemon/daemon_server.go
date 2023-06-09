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
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core/namegen"
	"github.com/rprtr258/pm/internal/infra/db"
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

func getProcCwd(cwd, procCwd string) string {
	if filepath.IsAbs(procCwd) {
		return procCwd
	}

	return filepath.Join(cwd, procCwd)
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
		procIDStr := strconv.FormatUint(uint64(proc.ProcID), 10) //nolint:gomnd // decimal
		logsDir := path.Join(srv.homeDir, "logs")

		if err := os.Mkdir(logsDir, os.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, xerr.NewWM(err, "create logs dir", xerr.Fields{"dir": logsDir})
		}

		stdoutLogFilename := path.Join(logsDir, procIDStr+".stdout")
		stdoutLogFile, err := os.OpenFile(stdoutLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
		if err != nil {
			return nil, xerr.NewWM(err, "open stdout file", xerr.Fields{"filename": stdoutLogFile})
		}
		defer stdoutLogFile.Close()

		stderrLogFilename := path.Join(logsDir, procIDStr+".stderr")
		stderrLogFile, err := os.OpenFile(stderrLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
		if err != nil {
			return nil, xerr.NewWM(err, "open stderr file", xerr.Fields{"filename": stderrLogFilename})
		}
		defer stderrLogFile.Close() // TODO: wrap
		// TODO: syscall.CloseOnExec(pidFile.Fd()) or just close pid file

		cwd, err := os.Getwd()
		if err != nil {
			return nil, xerr.NewWM(err, "os.Getwd")
		}

		procAttr := os.ProcAttr{
			Dir: getProcCwd(cwd, proc.Cwd),
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
			if errSetStatus := srv.db.SetStatus(proc.ProcID, db.NewStatusInvalid()); errSetStatus != nil {
				return nil, xerr.NewWM(xerr.Combine(err, errSetStatus), "running failed, setting errored status failed")
			}

			return nil, xerr.NewWM(err, "running failed", xerr.Fields{"procData": spew.Sprint(proc)})
		}

		runningStatus := db.NewStatusRunning(time.Now(), process.Pid, 0, 0)
		if err := srv.db.SetStatus(proc.ProcID, runningStatus); err != nil {
			return nil, xerr.NewWM(err, "set status running", xerr.Fields{"procID": proc.ProcID})
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
		xerr.AppendInto(&merr, srv.stop(proc))
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
	process, errFindProc := os.FindProcess(proc.Status.Pid)
	if errFindProc != nil {
		return xerr.NewWM(errFindProc, "getting process by pid failed", xerr.Fields{"pid": proc.Status.Pid})
	}

	// TODO: kill after timeout
	if errKill := syscall.Kill(-process.Pid, syscall.SIGTERM); errKill != nil {
		if errors.Is(errKill, os.ErrProcessDone) {
			log.Printf("[WARN] finished process %+v with running status", proc)
		} else {
			return xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": process.Pid})
		}
	}

	state, errFindProc := process.Wait()
	if errFindProc != nil {
		if errno, ok := xerr.As[syscall.Errno](errFindProc); !ok || errno != 10 {
			return xerr.NewWM(errFindProc, "releasing process", xerr.Fields{"procData": proc})
		}

		fmt.Printf("[INFO] process %+v is not a child", proc)
	} else {
		fmt.Printf("[INFO] process %+v closed with state %+v\n", proc, state)
	}

	if errSetStatus := srv.db.SetStatus(proc.ProcID, db.NewStatusStopped(state.ExitCode())); errSetStatus != nil {
		return status.Errorf(codes.DataLoss, "set status stopped, procID=%d: %s", proc.ProcID, errSetStatus.Error())
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
				ProcID:  procID,
				Status:  db.NewStatusStarting(),
				Name:    procOpts.GetName(),
				Cwd:     procOpts.GetCwd(),
				Tags:    lo.Uniq(append(procOpts.GetTags(), "all")),
				Command: procOpts.GetCommand(),
				Args:    procOpts.GetArgs(),
				Watch:   nil,
			}

			if errUpdate := srv.db.UpdateProc(procData); errUpdate != nil {
				return nil, xerr.NewWM(errUpdate, "update proc", xerr.Fields{"procData": procData})
			}

			return &api.ProcessID{Id: uint64(procID)}, nil
		}
	}

	name := lo.IfF(procOpts.Name != nil, procOpts.GetName).ElseF(namegen.New)

	procData := db.ProcData{
		ProcID:  0, // TODO: create instead proc create query
		Status:  db.NewStatusStarting(),
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

//nolint:exhaustruct // can't return api.isProcessStatus_Status
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
	if errDelete := srv.db.Delete(ids); errDelete != nil {
		return nil, xerr.NewWM(errDelete, "delete proc", xerr.Fields{"procIDs": ids})
	}
	// TODO: add loggings

	var merr error
	for _, procID := range ids {
		if err := removeLogFiles(procID); err != nil {
			xerr.AppendInto(&merr, xerr.NewWM(err, "delete proc", xerr.Fields{"procID": procID}))
		}
	}

	return &emptypb.Empty{}, merr
}

func removeLogFiles(procID uint64) error {
	stdoutFilename := filepath.Join(_daemonLogsDir, fmt.Sprintf("%d.stdout", procID))
	if errRmStdout := removeFile(stdoutFilename); errRmStdout != nil {
		return errRmStdout
	}

	stderrFilename := filepath.Join(_daemonLogsDir, fmt.Sprintf("%d.stderr", procID))
	if errRmStderr := removeFile(stderrFilename); errRmStderr != nil {
		return errRmStderr
	}

	return nil
}

func removeFile(name string) error {
	if _, errStat := os.Stat(name); errStat != nil {
		if os.IsNotExist(errStat) {
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
