package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/db"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

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

	// TODO: if ~home/logs does not exist - create

	for _, proc := range procs {
		procIDStr := strconv.FormatUint(uint64(proc.ID), 10)
		logsDir := path.Join(srv.homeDir, "logs")

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
		defer stderrLogFile.Close()

		execCmd := exec.CommandContext(ctx, "/usr/bin/bash", []string{"-c", proc.Cmd}...)
		execCmd.Stdout = stdoutLogFile
		execCmd.Stderr = stderrLogFile

		if err := db.New(srv.dbFile).SetStatus(proc.ID, db.StatusRunning); err != nil {
			return nil, err
		}

		// TODO: run in goroutine/syscall.ForkExec()/os.StartProcess
		// wd, _ := os.Getwd()
		// proc := &os.ProcAttr{
		// 	Dir: wd,
		// 	Env: os.Environ(),
		// 	Files: []*os.File{
		// 		os.Stdin,
		// 		NewLog(p.Logfile),
		// 		NewLog(p.Errfile),
		// 	},
		// }
		// args := append([]string{p.Name}, p.Args...)
		// process, err := os.StartProcess(p.Command, args, proc)
		// process.Pid
		if err := execCmd.Run(); err != nil {
			if err2 := db.New(srv.dbFile).SetStatus(proc.ID, db.StatusErrored); err2 != nil {
				return nil, fmt.Errorf("running failed: %w; setting errored status failed: %w", err, err2)
			}

			return nil, fmt.Errorf("running failed: %w", err)
		}

		if err := db.New(srv.dbFile).SetStatus(proc.ID, db.StatusStopped); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

// Stop - stop processes by their ids in database
// TODO: change to sending signals
func (srv *daemonServer) Stop(_ context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	dbHandle := db.New(srv.dbFile)

	procsToStop := req.GetIds()

	for _, id := range procsToStop {
		// TODO: actually stop proc
		// os.FindProcess()
		// proc.Release()
		if err := dbHandle.SetStatus(db.ProcID(id), db.StatusStopped); err != nil {
			return nil, status.Errorf(codes.DataLoss, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}
