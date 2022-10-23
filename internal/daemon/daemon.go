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

func encodeUintKey(procID uint64) []byte {
	return []byte(strconv.FormatUint(procID, 10))
}

func decodeUintKey(key []byte) (uint64, error) {
	return strconv.ParseUint(string(key), 10, 64)
}

func encodeJSON[T any](proc T) ([]byte, error) {
	return json.Marshal(proc)
}

func decodeJSON[T any](value []byte) (T, error) {
	var res T
	if err := json.Unmarshal(value, &res); err != nil {
		return lo.Empty[T](), fmt.Errorf("failed decoding value of type %T: %w", res, err)
	}

	return res, nil
}

// TODO: template key type
func get[V any](bucket *bbolt.Bucket, key []byte) (V, error) {
	// keyBytes, err := encodeUintKey(key)
	bytes := bucket.Get(key)
	if bytes == nil {
		return lo.Empty[V](), fmt.Errorf("value not found by key %v", key)
	}

	return decodeJSON[V](bytes)
}

// TODO: serialize/deserialize protobuffers
// TODO: template key type
func put[V any](bucket *bbolt.Bucket, key []byte, value V) error {
	bytes, err := encodeJSON(value)
	if err != nil {
		return err
	}

	if err := bucket.Put(key, bytes); err != nil {
		return err
	}

	return nil
}

func (db *DB) AddTask(metadata ProcMetadata) (uint64, error) {
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

			if err := put(mainBucket, encodeUintKey(id), metadata); err != nil {
				return err
			}
		}
		{
			byNameBucket := tx.Bucket([]byte(_byNameBucket))
			if byNameBucket == nil {
				return errors.New("byName bucket was not found")
			}

			idsByName, err := get[[]uint64](byNameBucket, []byte(metadata.Name))
			if err != nil {
				return err
			}

			if err := put(byNameBucket, []byte(metadata.Name), append(idsByName, procID)); err != nil {
				return err
			}
		}
		{
			byTagBucket := tx.Bucket([]byte(_byTagBucket))
			if byTagBucket == nil {
				return errors.New("byTag bucket was not found")
			}

			for _, tag := range metadata.Tags {
				idsByTag, err := get[[]uint64](byTagBucket, []byte(tag))
				if err != nil {
					return err
				}

				if err := put(byTagBucket, []byte(metadata.Name), append(idsByTag, procID)); err != nil {
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
			id, err := decodeUintKey(key)
			if err != nil {
				return fmt.Errorf("incorrect key: %w", err)
			}

			metadata, err := decodeJSON[ProcMetadata](value)
			if err != nil {
				return err
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

func (db *DB) SetStatus(procID uint64, newStatus string /*TODO: enum statuses*/) error {
	return db.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(_mainBucket))
		if bucket == nil {
			return errors.New("main bucket does not exist")
		}

		key := encodeUintKey(procID)
		metadata, err := get[ProcMetadata](bucket, key)
		if err != nil {
			return err
		}

		metadata.Status = newStatus

		return put(bucket, key, metadata)
	})
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

func (srv *daemonServer) Stop(_ context.Context, req *pb.DeleteReq) (*pb.DeleteResp, error) {
	procs, err := srv.DB.List()
	if err != nil {
		return nil, status.Errorf(codes.DataLoss, err.Error())
	}

	procsToStop := lo.Filter(procs, func(proc ProcData, _ int) bool {
		// TODO: filter based on req.GetFilters()
		return true
	})

	for _, proc := range procsToStop {
		// TODO: stop proc
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

func (*daemonServer) Delete(_ context.Context, req *pb.DeleteReq) (*pb.DeleteResp, error) {
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
