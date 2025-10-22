package core

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

const EnvPMID = "PM_PMID"

var (
	DirHome     = filepath.Join(xdg.DataHome, "pm")
	DirLogs     = filepath.Join(DirHome, "logs")
	DirDB       = filepath.Join(DirHome, "db")
	_configPath = filepath.Join(xdg.ConfigHome, "pm.json")
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
}
