package core

import (
	"os"
	"path/filepath"

	"github.com/rprtr258/log"
)

var (
	_userHome       = userHomeDir()
	DirHome         = filepath.Join(_userHome, ".pm")
	DirDaemonLogs   = filepath.Join(DirHome, "logs")
	FileDaemonPid   = filepath.Join(DirHome, "pm.pid")
	FileDaemonLog   = filepath.Join(DirHome, "pm.log")
	FileDaemonDBDir = filepath.Join(DirHome, "db")
	SocketDaemonRPC = filepath.Join(DirHome, "rpc.sock")
)

func userHomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("can't get home dir", log.F{"err": err.Error()})
	}

	return dir
}
