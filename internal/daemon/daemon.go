package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

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

const (
	_mainBucket   = "main"
	_byNameBucket = "by_name"
	_byTagBucket  = "by_tag"
)

type ProcMetadata struct {
	Name   string   `json:"name"`
	Cmd    string   `json:"cmd"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
}

type DB struct {
	db bbolt.DB
}

func (db *DB) AddTask(metadata ProcMetadata) (uint64, error) {
	// TODO: serialize/deserialize protobuffers
	encodedMetadata, err := json.Marshal(metadata)
	if err != nil {
		return 0, err
	}

	var procID uint64
	if err := db.db.Update(func(tx *bbolt.Tx) error {
		{
			mainBucket := tx.Bucket([]byte(_mainBucket))
			if mainBucket == nil {
				return errors.New("main bucket was not found")
			}

			id, err := mainBucket.NextSequence()
			if err != nil {
				return err
			}

			procID = id

			idBytes := []byte(strconv.FormatInt(int64(procID), 10))
			if err := mainBucket.Put(idBytes, encodedMetadata); err != nil {
				return err
			}
		}
		{
			byNameBucket := tx.Bucket([]byte(_byNameBucket))
			if byNameBucket == nil {
				return errors.New("byName bucket was not found")
			}

			idsByNameBytes := byNameBucket.Get([]byte(metadata.Name))
			var idsByName []int64
			if idsByName != nil {
				if err := json.Unmarshal(idsByNameBytes, &idsByName); err != nil {
					return err
				}
			}

			newIdsByName := append(idsByName, int64(procID))
			newIdsByNameBytes, err := json.Marshal(newIdsByName)
			if err != nil {
				return err
			}

			if err := byNameBucket.Put([]byte(metadata.Name), newIdsByNameBytes); err != nil {
				return err
			}
		}
		{
			byTagBucket := tx.Bucket([]byte(_byTagBucket))
			if byTagBucket == nil {
				return errors.New("byTag bucket was not found")
			}

			for _, tag := range metadata.Tags {
				idsWithSuchNameBytes := byTagBucket.Get([]byte(tag))
				var idsByTag []int64
				if idsWithSuchNameBytes != nil {
					if err := json.Unmarshal(idsWithSuchNameBytes, &idsByTag); err != nil {
						return err
					}
				}

				newIdsWithSuchName := append(idsByTag, int64(procID))
				newIdsWithSuchNameBytes, err := json.Marshal(newIdsWithSuchName)
				if err != nil {
					return err
				}

				if err := byTagBucket.Put([]byte(metadata.Name), newIdsWithSuchNameBytes); err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return procID, nil
}

type ProcData struct {
	ID       uint64
	Metadata ProcMetadata
}

func (db *DB) List() ([]ProcData, error) {
	var res []ProcData

	if err := db.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(_mainBucket))
		if bucket == nil {
			return errors.New("main bucket does not exist")
		}

		if err := bucket.ForEach(func(key, value []byte) error {
			id, err := strconv.ParseUint(string(key), 10, 64)
			if err != nil {
				return fmt.Errorf("incorrect key: %w", err)
			}

			var metadata ProcMetadata
			if err := json.Unmarshal(value, &metadata); err != nil {
				return fmt.Errorf("failed decoding value: %w", err)
			}

			res = append(res, ProcData{
				ID: id,
				Metadata: ProcMetadata{
					Name:   metadata.Name,
					Cmd:    metadata.Cmd,
					Status: metadata.Status,
					Tags:   metadata.Tags,
				},
			})

			return nil
		}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

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

func (*daemonServer) Stop(context.Context, *pb.DeleteReq) (*pb.DeleteResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}

func (*daemonServer) Delete(context.Context, *pb.DeleteReq) (*pb.DeleteResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}

// Run daemon
func Run(rpcSocket, dbFile string) error {
	sock, err := net.Listen("unix", rpcSocket)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer sock.Close()

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
