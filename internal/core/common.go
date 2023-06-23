package core

import (
	"os"
	"path/filepath"

	"golang.org/x/exp/slog"
)

var (
	_userHome = userHomeDir()
	DirHome   = filepath.Join(_userHome, ".pm")
	SocketRPC = filepath.Join(DirHome, "rpc.sock")
)

func userHomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("can't get home dir", "err", err.Error())
		os.Exit(1)
	}

	return dir
}
