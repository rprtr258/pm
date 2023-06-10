package core

import (
	"os"
	"path/filepath"
)

var (
	_userHome       = os.Getenv("HOME")
	DirHome         = filepath.Join(_userHome, ".pm")
	DirDaemonLogs   = filepath.Join(DirHome, "logs")
	FileDaemonPid   = filepath.Join(DirHome, "pm.pid")
	FileDaemonLog   = filepath.Join(DirHome, "pm.log")
	FileDaemonDBDir = filepath.Join(DirHome, "db")
	SocketDaemonRPC = filepath.Join(DirHome, "rpc.sock")
)
