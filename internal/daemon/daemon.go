package daemon

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/samber/lo"
	"github.com/sevlyar/go-daemon"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
	DB *DB
}

// TODO: use grpc status codes
func (srv *daemonServer) Start(ctx context.Context, req *pb.StartReq) (*pb.StartResp, error) {
	// startParamsProto := req.GetProcess()
	// TODO: move to client
	// switch /*startParams :=*/ startParamsProto.(type) {
	// case *pb.StartReq_Cmd:
	// 	stdoutLogFile, err := os.OpenFile(path.Join(HomeDir, name, "stdout"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	defer stdoutLogFile.Close()

	// 	stderrLogFile, err := os.OpenFile(path.Join(HomeDir, name, "stderr"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	defer stderrLogFile.Close()

	// 	pidFile, err := os.OpenFile(path.Join(HomeDir, name, "pid"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	defer pidFile.Close()

	// 	// TODO: syscall.ForkExec()
	// 	execCmd := exec.CommandContext(context.TODO(), cmd, startParamsProto...)
	// 	execCmd.Stdout = stdoutLogFile
	// 	execCmd.Stderr = stderrLogFile
	// 	if err := execCmd.Start(); err != nil {
	// 		return nil, err
	// 	}

	// 	if _, err := pidFile.WriteString(strconv.Itoa(execCmd.Process.Pid)); err != nil {
	// 		return nil, err
	// 	}
	// // Processes[name] = execCmd.Process.Pid
	// return &pb.StartResp{
	// 	Id:  0,
	// 	Pid: int64(execCmd.Process.Pid),
	// }, nil
	// case *pb.StartReq_Shell:
	// case *pb.StartReq_Config:
	// }

	metadata := ProcMetadata{
		Name:   req.GetName(),
		Cmd:    req.GetCmd(),
		Status: "running",
		Tags:   lo.Uniq(append(req.GetTags().GetTags(), "all")),
	}

	procID, err := srv.DB.AddTask(metadata)
	if err != nil {
		return nil, err
	}

	return &pb.StartResp{
		Id:  procID,
		Pid: 1,
	}, nil
}

func mapStatus(status string) func(*pb.ListRespEntry) {
	switch status {
	case "running":
		return func(lre *pb.ListRespEntry) {
			lre.Status = &pb.ListRespEntry_Running{
				Running: &pb.RunningInfo{
					Pid:    0,
					Uptime: durationpb.New(0),
				},
			}
		}
	case "stopped":
		return func(lre *pb.ListRespEntry) {
			lre.Status = &pb.ListRespEntry_Stopped{}
		}
	case "errored":
		return func(lre *pb.ListRespEntry) {
			lre.Status = &pb.ListRespEntry_Errored{}
		}
	default:
		return func(lre *pb.ListRespEntry) {
			lre.Status = &pb.ListRespEntry_Invalid{Invalid: status}
		}
	}
}

func (srv *daemonServer) List(context.Context, *emptypb.Empty) (*pb.ListResp, error) {
	resp, err := srv.DB.List()
	if err != nil {
		return nil, err
	}

	return &pb.ListResp{
		Items: lo.Map(resp, func(elem ProcData, _ int) *pb.ListRespEntry {
			res := &pb.ListRespEntry{
				Id:     elem.ID,
				Cmd:    elem.Metadata.Cmd,
				Name:   elem.Metadata.Name,
				Tags:   &pb.Tags{Tags: elem.Metadata.Tags},
				Cpu:    0, // TODO: take from ps
				Memory: 0, // TODO: take from ps
			}
			mapStatus(elem.Metadata.Status)(res)
			return res
		}),
	}, nil
}

func filterProcs(db *DB) ([]ProcData, error) {
	procs, err := db.List()
	if err != nil {
		return nil, status.Errorf(codes.DataLoss, err.Error())
	}

	return lo.Filter(procs, func(proc ProcData, _ int) bool {
		// TODO: filter based on req.GetFilters()
		return true
	}), nil
}

func (srv *daemonServer) Stop(_ context.Context, req *pb.DeleteReq) (*pb.DeleteResp, error) {
	procsToStop, err := filterProcs(srv.DB)
	if err != nil {
		return nil, status.Errorf(codes.DataLoss, err.Error())
	}

	for _, proc := range procsToStop {
		// TODO: stop proc
		// TODO: batch
		if err := srv.DB.SetStatus(proc.ID, "stopped"); err != nil {
			return nil, status.Errorf(codes.DataLoss, err.Error())
		}
	}

	return &pb.DeleteResp{
		Id: lo.Map(procsToStop, func(proc ProcData, _ int) uint64 {
			return proc.ID
		}),
	}, nil
}

func (srv *daemonServer) Delete(_ context.Context, req *pb.DeleteReq) (*pb.DeleteResp, error) {
	procsToDelete, err := filterProcs(srv.DB)
	if err != nil {
		return nil, status.Errorf(codes.DataLoss, err.Error())
	}

	for _, proc := range procsToDelete {
		// TODO: batch
		if err := srv.DB.Delete(proc.ID); err != nil {
			return nil, err
		}
	}

	return &pb.DeleteResp{
		Id: lo.Map(procsToDelete, func(proc ProcData, _ int) uint64 {
			return proc.ID
		}),
	}, nil
}

// Run daemon
func Run(rpcSocket, dbFile string) error {
	sock, err := net.Listen("unix", rpcSocket)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer sock.Close()

	// TODO: open on every request, don't hold lock/rewrite to sqlite
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(_mainBucket)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(_byNameBucket)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(_byTagBucket)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	srv := grpc.NewServer()
	pb.RegisterDaemonServer(srv, &daemonServer{
		DB: &DB{db: *db},
	})

	log.Printf("daemon started at %v", sock.Addr())
	if err := srv.Serve(sock); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Kill daemon
func Kill(daemonCtx *daemon.Context, rpcSocket string) error {
	if err := os.Remove(rpcSocket); err != nil {
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
