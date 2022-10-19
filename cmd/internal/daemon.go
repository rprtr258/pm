package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/sevlyar/go-daemon"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
)

type daemonServer struct {
	pb.UnimplementedDaemonServer
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

func (*daemonServer) List(context.Context, *emptypb.Empty) (*pb.ListResp, error) {
	fs, err := os.ReadDir(HomeDir)
	if err != nil {
		return nil, err
	}

	for _, f := range fs {
		if !f.IsDir() {
			// fmt.Fprintf(os.Stderr, "found strange file %q which should not exist\n", path.Join(HomeDir, f.Name()))
			continue
		}

		// fmt.Printf("%#v", f.Name())
	}

	return &pb.ListResp{
		Items: []*pb.ListRespEntry{},
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
		log.Fatalf("failed to listen: %v", err)
	}
	defer sock.Close()

	srv := grpc.NewServer()
	pb.RegisterDaemonServer(srv, &daemonServer{})

	log.Println("- - - - - - - - - - - - - - -")
	log.Printf("daemon started at %v", sock.Addr())
	if err := srv.Serve(sock); err != nil {
		log.Fatalf("failed to serve: %v", err)
		return err
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
