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

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/db"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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
	procs, err := db.New(srv.dbFile).GetProcs(lo.Map(req.GetIds(), func(id uint64, _ int) db.ProcID {
		return db.ProcID(id)
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

		cwd, err := os.Getwd() // TODO: ???
		if err != nil {
			return nil, fmt.Errorf("os.Getwd failed: %w", err)
		}

		procAttr := os.ProcAttr{
			Dir: cwd,
			Env: os.Environ(), // TODO: ???
			Files: []*os.File{
				os.Stdin,
				stdoutLogFile,
				stderrLogFile,
			},
			Sys: &syscall.SysProcAttr{},
		}
		// args := append([]string{p.Name}, p.Args...)

		process, err := os.StartProcess(proc.Command, append([]string{proc.Command}, proc.Args...), &procAttr)
		if err != nil {
			if err2 := db.New(srv.dbFile).SetStatus(proc.ID, db.Status{Status: db.StatusErrored}); err2 != nil {
				return nil, fmt.Errorf("running failed: %w; setting errored status failed: %w", err, err2)
			}

			return nil, fmt.Errorf("running failed: %w", err)
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

	procsToStop := lo.Map(req.GetIds(), func(id uint64, _ int) db.ProcID {
		return db.ProcID(id)
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

		if err := process.Kill(); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				log.Printf("[WARN] finished process %+v with running status", proc)
			} else {
				return nil, fmt.Errorf("killing process with pid=%d failed: %w", process.Pid, err)
			}
		}

		if err := dbHandle.SetStatus(proc.ID, db.Status{Status: db.StatusStopped}); err != nil {
			return nil, status.Errorf(codes.DataLoss, fmt.Errorf("updating status of process %+v failed: %w", proc, err).Error())
		}
	}

	return &emptypb.Empty{}, nil
}
