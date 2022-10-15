package internal

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/sevlyar/go-daemon"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"

	pb "github.com/rprtr258/pm/api"
)

type daemonServer struct {
	pb.UnimplementedGreeterServer
}

func (*daemonServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Println("HI", req.GetName())
	return &pb.HelloReply{
		Message: req.GetName(),
	}, nil
}

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "run daemon",
	Action: func(*cli.Context) error {
		daemonCtx := &daemon.Context{
			PidFileName: DaemonPidFile,
			PidFilePerm: 0644,
			LogFileName: DaemonLogFile,
			LogFilePerm: 0640,
			WorkDir:     "./",
			Umask:       027,
			Args:        []string{"pm", "daemon"},
		}
		d, err := daemonCtx.Reborn()
		if err != nil {
			return fmt.Errorf("unable to run: %w", err)
		}

		if d != nil {
			fmt.Println(d.Pid)
			return nil
		}

		defer daemonCtx.Release()

		sock, err := net.Listen("unix", DaemonRpcSocket)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		srv := grpc.NewServer()
		pb.RegisterGreeterServer(srv, &daemonServer{})

		log.Println("- - - - - - - - - - - - - - -")
		log.Printf("daemon started at %v", sock.Addr())
		if err := srv.Serve(sock); err != nil {
			log.Fatalf("failed to serve: %v", err)
			return err
		}

		return nil
	},
}
