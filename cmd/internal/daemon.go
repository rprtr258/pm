package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/sevlyar/go-daemon"
	"github.com/urfave/cli/v2"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
)

const (
	mainBucket   = "main"
	byNameBucket = "by_name"
	byTagBucket  = "by_tag"
)

type ProcMetadata struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
}

type daemonServer struct {
	pb.UnimplementedDaemonServer
	DB *bbolt.DB
}

func (*daemonServer) Start(ctx context.Context, req *pb.StartReq) (*pb.StartResp, error) {
	// name := req.GetName()
	// cmd := req.GetCmd()
	startParamsProto := req.GetProcess()
	switch /*startParams :=*/ startParamsProto.(type) {
	case *pb.StartReq_Cmd:
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
	case *pb.StartReq_Shell:
	case *pb.StartReq_Config:
	}

	return &pb.StartResp{
		Id:  0,
		Pid: 1,
	}, nil
}

func (srv *daemonServer) List(context.Context, *emptypb.Empty) (*pb.ListResp, error) {
	resp := []*pb.ListRespEntry{}

	err := srv.DB.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(mainBucket))
		if bucket == nil {
			return errors.New("main bucket does not exist")
		}

		if err := bucket.ForEach(func(key, value []byte) error {
			id, err := strconv.ParseInt(string(key), 10, 64)
			if err != nil {
				return fmt.Errorf("incorrect key: %w", err)
			}

			var metadata ProcMetadata
			if err := json.Unmarshal(value, &metadata); err != nil {
				return fmt.Errorf("failed decoding value: %w", err)
			}

			resp = append(resp, &pb.ListRespEntry{
				Id:     id,
				Name:   metadata.Name,
				Status: &pb.ListRespEntry_Errored{},
				Tags:   &pb.Tags{Tags: metadata.Tags},
				Cpu:    0, // TODO: take from ps
				Memory: 0, // TODO: take from ps
			})

			return nil
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.ListResp{
		Items: resp,
	}, nil
}

func (*daemonServer) Stop(context.Context, *pb.DeleteReq) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}

func (*daemonServer) Delete(context.Context, *pb.DeleteReq) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}

func init() {
	AllCmds = append(AllCmds, DaemonCmd)
}

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "manage daemon",
	Subcommands: []*cli.Command{
		{
			Name:  "start",
			Usage: "launch daemon process",
			Action: func(ctx *cli.Context) error {
				daemonCtx := &daemon.Context{
					PidFileName: DaemonPidFile,
					PidFilePerm: 0644,
					LogFileName: DaemonLogFile,
					LogFilePerm: 0640,
					WorkDir:     "./",
					Umask:       027,
					Args:        []string{"pm", "daemon", "start"},
				}

				killDaemon(daemonCtx)

				d, err := daemonCtx.Reborn()
				if err != nil {
					return fmt.Errorf("unable to run: %w", err)
				}

				if d != nil {
					fmt.Println(d.Pid)
					return nil
				}

				defer daemonCtx.Release()

				return runDaemon()
			},
		},
		{
			Name:  "stop",
			Usage: "stop daemon process",
			Action: func(ctx *cli.Context) error {
				daemonCtx := &daemon.Context{
					PidFileName: DaemonPidFile,
					PidFilePerm: 0644,
					LogFileName: DaemonLogFile,
					LogFilePerm: 0640,
					WorkDir:     "./",
					Umask:       027,
					Args:        []string{"pm", "daemon", "start"},
				}

				killDaemon(daemonCtx)
				return nil
			},
		},
		{
			Name:  "run",
			Usage: "run daemon, DON'T USE BY HAND IF YOU DON'T KNOW WHAT YOU ARE DOING",
			Action: func(ctx *cli.Context) error {
				return runDaemon()
			},
		},
	},
}

func runDaemon() error {
	sock, err := net.Listen("unix", DaemonRpcSocket)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer sock.Close()

	db, err := bbolt.Open(DBFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(mainBucket)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(byNameBucket)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(byTagBucket)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	srv := grpc.NewServer()
	pb.RegisterDaemonServer(srv, &daemonServer{
		DB: db,
	})

	log.Printf("daemon started at %v", sock.Addr())
	if err := srv.Serve(sock); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func killDaemon(daemonCtx *daemon.Context) {
	if err := os.Remove(DaemonRpcSocket); err != nil {
		log.Println("error removing socket file:", err.Error())
	}

	if proc, err := daemonCtx.Search(); err != nil {
		log.Println("failed searching daemon:", err.Error())
	} else if proc != nil {
		if err := proc.Kill(); err != nil {
			log.Println("failed killing daemon:", err.Error())
		}
	}
}
