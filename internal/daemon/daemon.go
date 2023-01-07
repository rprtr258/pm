package daemon

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/pm/internal/go-daemon"
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

	dbHandle := db.New(dbFile)

	if err := dbHandle.Init(); err != nil {
		return err
	}

	srv := grpc.NewServer()
	api.RegisterDaemonServer(srv, &daemonServer{
		db:      dbHandle,
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
	// TODO: rewrite checking pid file not found error
	if err != nil && err.Error() != "open /home/rprtr258/.pm/pm.pid: no such file or directory" {
		return fmt.Errorf("searching daemon failed: %w", err)
	}

	if proc == nil {
		return nil
	}

	for {
		if err := proc.Kill(); err != nil {
			if err == os.ErrProcessDone {
				break
			}
			return fmt.Errorf("killing daemon process failed: %w", err)
		}
	}

	return nil
}
