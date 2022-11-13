package daemon

import (
	"context"
	"fmt"
	"os"
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
		}
		// args := append([]string{p.Name}, p.Args...)
		// process, err := os.StartProcess(p.Command, args, proc)
		// process.Pid
		// TODO: find bash
		_ /*process*/, err /*:*/ = os.StartProcess("/usr/bin/bash", []string{"/usr/bin/bash", "-c", proc.Cmd}, &procAttr)
		if err != nil {
			if err2 := db.New(srv.dbFile).SetStatus(proc.ID, db.StatusErrored); err2 != nil {
				return nil, fmt.Errorf("running failed: %w; setting errored status failed: %w", err, err2)
			}

			return nil, fmt.Errorf("running failed: %w", err)
		}

		// TODO: add pid
		if err := db.New(srv.dbFile).SetStatus(proc.ID, db.StatusRunning); err != nil {
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
