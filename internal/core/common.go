package core

import (
	"cmp"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const EnvPMID = "PM_PMID"

var _userHome = func() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		log.Error().Err(err).Msg("can't get home dir")
		os.Exit(1)
	}

	return dir
}()

var (
	DirHome       = cmp.Or(os.Getenv("PM_HOME"), filepath.Join(_userHome, ".pm"))
	_dirProcsLogs = filepath.Join(DirHome, "logs")
	_configPath   = filepath.Join(DirHome, "config.json")
	_dirDB        = filepath.Join(DirHome, "db")
)

type LogType int

const (
	LogTypeUnspecified LogType = iota
	LogTypeStdout
	LogTypeStderr
)

type LogLine struct {
	ProcID   PMID
	ProcName string
	Type     LogType
	Line     string
	Err      error
}
