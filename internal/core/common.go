package core

import (
	"os"
	"path/filepath"

	"github.com/rprtr258/log"
)

var (
	_userHome = userHomeDir()
	DirHome   = filepath.Join(_userHome, ".pm")
	SocketRPC = filepath.Join(DirHome, "rpc.sock")
)

func userHomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("can't get home dir", log.F{"err": err.Error()})
	}

	return dir
}
