package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/samber/lo"
	"github.com/sevlyar/go-daemon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	dbFile  string
	homeDir string
}

// TODO: use grpc status codes
func (srv *daemonServer) Start(ctx context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	db, err := New(srv.dbFile)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	procs, err := db.GetProcs(lo.Map(req.GetIds(), func(id uint64, _ int) ProcID {
		return ProcID(id)
	}))
	if err != nil {
		return nil, err
	}

	for _, proc := range procs {
		procIDStr := strconv.FormatUint(uint64(proc.ID), 10)
		logsDir := path.Join(srv.homeDir, "logs")

		stdoutLogFile, err := os.OpenFile(path.Join(logsDir, procIDStr+".stdout"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}
		defer stdoutLogFile.Close()

		stderrLogFile, err := os.OpenFile(path.Join(logsDir, procIDStr, "stderr"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}
		defer stderrLogFile.Close()

		// TODO: syscall.ForkExec()
		execCmd := exec.CommandContext(ctx, "/usr/bin/bash", []string{"-c", proc.Cmd}...)
		execCmd.Stdout = stdoutLogFile
		execCmd.Stderr = stderrLogFile

		if err := execCmd.Start(); err != nil {
			return nil, err
		}
	}

	fmt.Println(procs)

	return &emptypb.Empty{}, nil
}

// func mapStatus(status _status) func(*pb.ListRespEntry) {
// 	switch status {
// 	case _statusRunning:
// 		return func(lre *pb.ListRespEntry) {
// 			lre.Status = &pb.ListRespEntry_Running{
// 				Running: &pb.RunningInfo{
// 					Pid:    0,
// 					Uptime: durationpb.New(0),
// 				},
// 			}
// 		}
// 	case _statusStopped:
// 		return func(lre *pb.ListRespEntry) {
// 			lre.Status = &pb.ListRespEntry_Stopped{}
// 		}
// 	case _statusErrored:
// 		return func(lre *pb.ListRespEntry) {
// 			lre.Status = &pb.ListRespEntry_Errored{}
// 		}
// 	default:
// 		return func(lre *pb.ListRespEntry) {
// 			lre.Status = &pb.ListRespEntry_Invalid{Invalid: status}
// 		}
// 	}
// }

func (srv *daemonServer) Stop(_ context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	db, err := New(srv.dbFile)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	procsToStop := req.GetIds()

	for _, id := range procsToStop {
		// TODO: actually stop proc
		if err := db.SetStatus(id, StatusStopped); err != nil {
			return nil, status.Errorf(codes.DataLoss, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// TODO: fix "reborn failed: daemon: Resource temporarily unavailable" on start when
// daemon is already running
// Run daemon
func Run(rpcSocket, dbFile, homeDir string) error {
	sock, err := net.Listen("unix", rpcSocket)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer sock.Close()

	if err := DBInit(dbFile); err != nil {
		return err
	}

	srv := grpc.NewServer()
	pb.RegisterDaemonServer(srv, &daemonServer{
		dbFile:  dbFile,
		homeDir: homeDir,
	})

	log.Printf("daemon started at %v", sock.Addr())
	if err := srv.Serve(sock); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Kill daemon
func Kill(daemonCtx *daemon.Context, rpcSocket string) error {
	if err := os.Remove(rpcSocket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error removing socket file: %w", err)
	}

	proc, err := daemonCtx.Search()
	if err != nil {
		return fmt.Errorf("failed searching daemon: %w", err)
	}

	if proc != nil {
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("failed killing daemon: %w", err)
		}
	}

	return nil
}
