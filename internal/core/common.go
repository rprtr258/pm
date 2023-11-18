package core

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	_userHome = userHomeDir()
	DirHome   = filepath.Join(_userHome, ".pm")
	SocketRPC = filepath.Join(DirHome, "rpc.sock")
)

func userHomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		log.Error().Err(err).Msg("can't get home dir")
		os.Exit(1)
	}

	return dir
}

type LogType int

const (
	LogTypeUnspecified LogType = iota
	LogTypeStdout
	LogTypeStderr
)

type LogLine struct {
	ID   PMID
	Name string
	At   time.Time
	Type LogType
	Line string
	Err  error // TODO: also pass and process, e.g. for stopped proc
}
