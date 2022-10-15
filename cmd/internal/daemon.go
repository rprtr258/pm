package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/sevlyar/go-daemon"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"

	pb "github.com/rprtr258/pm/api"
)

type daemonServer struct {
	pb.UnimplementedGreeterServer
}

func (*daemonServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	// stdoutLogFile, err := os.OpenFile(path.Join(HomeDir, name, "stdout"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	// if err != nil {
	// 	return err
	// }
	// defer stdoutLogFile.Close()

	// stderrLogFile, err := os.OpenFile(path.Join(HomeDir, name, "stderr"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	// if err != nil {
	// 	return err
	// }
	// defer stderrLogFile.Close()

	// pidFile, err := os.OpenFile(path.Join(HomeDir, name, "pid"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	// if err != nil {
	// 	return err
	// }
	// defer pidFile.Close()

	// // TODO: syscall.ForkExec()
	// cmd := exec.CommandContext(ctx.Context, args[0], args[1:]...)
	// cmd.Stdout = stdoutLogFile
	// cmd.Stderr = stderrLogFile
	// if err := cmd.Start(); err != nil {
	// 	return err
	// }

	// if _, err := pidFile.WriteString(strconv.Itoa(cmd.Process.Pid)); err != nil {
	// 	return err
	// }

	// Processes[name] = cmd.Process.Pid

	// return nil
	log.Println("HI", req.GetName())
	return &pb.HelloReply{
		Message: req.GetName(),
	}, nil
}

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "manage daemon",
	Subcommands: []*cli.Command{
		{
			Name:  "start",
			Usage: "launch daemon process",
			Action: func(ctx *cli.Context) error {
				// killDaemon()

				// if err := os.Remove(DaemonPidFile); err != nil {
				// 	log.Println("error removing pid file:", err.Error())
				// }

				if err := os.Remove(DaemonRpcSocket); err != nil {
					log.Println("error removing socket file:", err.Error())
				}

				daemonCtx := &daemon.Context{
					PidFileName: DaemonPidFile,
					PidFilePerm: 0644,
					LogFileName: DaemonLogFile,
					LogFilePerm: 0640,
					WorkDir:     "./",
					Umask:       027,
					Args:        []string{"pm", "daemon", "start"},
				}

				if proc, err := daemonCtx.Search(); err != nil {
					log.Println("failed searching daemon:", err.Error())
				} else if proc != nil {
					if err := proc.Kill(); err != nil {
						log.Println("failed killing daemon:", err.Error())
					}
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

				return runDaemon()
			},
		},
		{
			Name:  "stop",
			Usage: "stop daemon process",
			Action: func(ctx *cli.Context) error {
				killDaemon()
				return nil
			},
		},
		{
			Name:  "run",
			Usage: "run daemon",
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
	pb.RegisterGreeterServer(srv, &daemonServer{})

	log.Println("- - - - - - - - - - - - - - -")
	log.Printf("daemon started at %v", sock.Addr())
	if err := srv.Serve(sock); err != nil {
		log.Fatalf("failed to serve: %v", err)
		return err
	}

	return nil
}

func killDaemon() {
	content, err := os.ReadFile(DaemonPidFile)
	if err != nil {
		log.Println("error reading pid file:", err.Error())
		return
	}

	pid, err := strconv.Atoi(string(content))
	if err != nil {
		log.Println("error parsing pid:", err.Error())
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		log.Println("error finding process:", err.Error())
		return
	}

	if err := process.Kill(); err != nil {
		log.Println("error killing process:", err.Error())
		return
	}
}
