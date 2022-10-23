package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/sevlyar/go-daemon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	dbFile string
}

// TODO: use grpc status codes
// func (srv *daemonServer) Start(ctx context.Context, req *pb.StartReq) (*pb.StartResp, error) {
// 	db, err := New(srv.dbFile)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer db.Close()

// 	// startParamsProto := req.GetProcess()
// 	// TODO: move to client
// 	// switch /*startParams :=*/ startParamsProto.(type) {
// 	// case *pb.StartReq_Cmd:
// 	// 	stdoutLogFile, err := os.OpenFile(path.Join(HomeDir, name, "stdout"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
// 	// 	if err != nil {
// 	// 		return nil, err
// 	// 	}
// 	// 	defer stdoutLogFile.Close()

// 	// 	stderrLogFile, err := os.OpenFile(path.Join(HomeDir, name, "stderr"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
// 	// 	if err != nil {
// 	// 		return nil, err
// 	// 	}
// 	// 	defer stderrLogFile.Close()

// 	// 	pidFile, err := os.OpenFile(path.Join(HomeDir, name, "pid"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
// 	// 	if err != nil {
// 	// 		return nil, err
// 	// 	}
// 	// 	defer pidFile.Close()

// 	// 	// TODO: syscall.ForkExec()
// 	// 	execCmd := exec.CommandContext(context.TODO(), cmd, startParamsProto...)
// 	// 	execCmd.Stdout = stdoutLogFile
// 	// 	execCmd.Stderr = stderrLogFile
// 	// 	if err := execCmd.Start(); err != nil {
// 	// 		return nil, err
// 	// 	}

// 	// 	if _, err := pidFile.WriteString(strconv.Itoa(execCmd.Process.Pid)); err != nil {
// 	// 		return nil, err
// 	// 	}
// 	// // Processes[name] = execCmd.Process.Pid
// 	// return &pb.StartResp{
// 	// 	Id:  0,
// 	// 	Pid: int64(execCmd.Process.Pid),
// 	// }, nil
// 	// case *pb.StartReq_Shell:
// 	// case *pb.StartReq_Config:
// 	// }

// 	metadata := ProcMetadata{
// 		Name: req.GetName(),
// 		Cmd:  req.GetCmd(),
// 		Status: Status{
// 			Status:    _statusRunning,
// 			Pid:       1,
// 			StartTime: time.Now(),
// 			Cpu:       0,
// 			Memory:    0,
// 		},
// 		Tags: lo.Uniq(append(req.GetTags().GetTags(), "all")),
// 	}

// 	procID, err := db.AddProc(metadata)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &pb.StartResp{
// 		Id:  procID,
// 		Pid: 1,
// 	}, nil
// }

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

// func (srv *daemonServer) List(context.Context, *emptypb.Empty) (*pb.ListResp, error) {
// 	db, err := New(srv.dbFile)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer db.Close()

// 	resp, err := db.List()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &pb.ListResp{
// 		Items: lo.Map(resp, func(elem ProcData, _ int) *pb.ListRespEntry {
// 			res := &pb.ListRespEntry{
// 				Id:     elem.ID,
// 				Cmd:    elem.Metadata.Cmd,
// 				Name:   elem.Metadata.Name,
// 				Tags:   &pb.Tags{Tags: elem.Metadata.Tags},
// 				Cpu:    0, // TODO: take from ps
// 				Memory: 0, // TODO: take from ps
// 			}
// 			mapStatus(elem.Metadata.Status)(res)
// 			return res
// 		}),
// 	}, nil
// }

// func filterProcs(db *DB) ([]ProcData, error) {
// 	procs, err := db.List()
// 	if err != nil {
// 		return nil, status.Errorf(codes.DataLoss, err.Error())
// 	}

// 	return lo.Filter(procs, func(proc ProcData, _ int) bool {
// 		// TODO: filter based on req.GetFilters()
// 		return true
// 	}), nil
// }

func (srv *daemonServer) Stop(_ context.Context, req *pb.IDs) (*emptypb.Empty, error) {
	db, err := New(srv.dbFile)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	procsToStop := req.GetIds()

	for _, id := range procsToStop {
		// TODO: stop proc
		// TODO: batch
		if err := db.SetStatus(id, StatusStopped); err != nil {
			return nil, status.Errorf(codes.DataLoss, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// func (srv *daemonServer) Delete(_ context.Context, req *pb.DeleteReq) (*pb.DeleteResp, error) {
// 	db, err := New(srv.dbFile)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer db.Close()

// 	procsToDelete, err := filterProcs(db)
// 	if err != nil {
// 		return nil, status.Errorf(codes.DataLoss, err.Error())
// 	}

// 	for _, proc := range procsToDelete {
// 		// TODO: batch
// 		if err := db.Delete(proc.ID); err != nil {
// 			return nil, err
// 		}
// 	}

// 	return &pb.DeleteResp{
// 		Id: lo.Map(procsToDelete, func(proc ProcData, _ int) uint64 {
// 			return proc.ID
// 		}),
// 	}, nil
// }

// TODO: fix "reborn failed: daemon: Resource temporarily unavailable" on start when
// daemon is already running
// Run daemon
func Run(rpcSocket, dbFile string) error {
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
		dbFile: dbFile,
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
