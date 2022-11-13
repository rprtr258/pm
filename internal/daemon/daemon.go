package daemon

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/sevlyar/go-daemon"
	"google.golang.org/grpc"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/db"
)

// TODO: fix "reborn failed: daemon: Resource temporarily unavailable" on start when
// daemon is already running
// Run daemon
func Run(rpcSocket, dbFile, homeDir string) error {
	sock, err := net.Listen("unix", rpcSocket)
	if err != nil {
		return fmt.Errorf("net.Listen failed: %w", err)
	}
	defer sock.Close()

	if err := db.New(dbFile).Init(); err != nil {
		return err
	}

	srv := grpc.NewServer()
	pb.RegisterDaemonServer(srv, &daemonServer{
		dbFile:  dbFile,
		homeDir: homeDir,
	})

	log.Printf("daemon started at %v", sock.Addr())
	if err := srv.Serve(sock); err != nil {
		return fmt.Errorf("serve failed: %w", err)
	}

	return nil
}

// Kill daemon
func Kill(daemonCtx *daemon.Context, rpcSocket string) error {
	if err := os.Remove(rpcSocket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing socket file failed: %w", err)
	}

	proc, err := daemonCtx.Search()
	if err != nil {
		return fmt.Errorf("searching daemon failed: %w", err)
	}

	if proc != nil {
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("killing daemon failed: %w", err)
		}
	}

	return nil
}
