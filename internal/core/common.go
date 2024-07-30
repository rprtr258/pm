package core

import (
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
	_dirHome      = filepath.Join(_userHome, ".pm")
	_dirProcsLogs = filepath.Join(_dirHome, "logs")
	_configPath   = filepath.Join(_dirHome, "config.json")
	_dirDB        = filepath.Join(_dirHome, "db")
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
