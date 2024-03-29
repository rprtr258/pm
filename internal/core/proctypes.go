package core

import (
	"fmt"
	"strconv"
	"time"
)

type StatusType int

const (
	StatusInvalid StatusType = iota
	StatusCreated
	StatusRunning
	StatusStopped
)

func (ps StatusType) String() string {
	switch ps {
	case StatusInvalid:
		return "invalid"
	case StatusCreated:
		return "created"
	case StatusRunning:
		return "running"
	case StatusStopped:
		return "stopped"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", ps)
	}
}

type Status struct {
	StartTime time.Time // StartTime, valid if running
	StoppedAt time.Time // StoppedAt - time when the process stopped, valid if stopped
	Status    StatusType
	Pid       int    // PID, valid if running
	CPU       uint64 // CPU usage percentage rounded to integer, valid if running
	Memory    uint64 // Memory usage in bytes, valid if running
	ExitCode  int    // ExitCode of the process, valid if stopped
}

func NewStatusInvalid() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusInvalid,
	}
}

func NewStatusCreated() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusCreated,
	}
}

func NewStatusRunning(startTime time.Time, pid int, cpu, memory uint64) Status {
	return Status{ //nolint:exhaustruct // not needed
		Status:    StatusRunning,
		StartTime: startTime,
		Pid:       pid,
		CPU:       cpu,
		Memory:    memory,
	}
}

func NewStatusStopped(exitCode int) Status {
	return Status{ //nolint:exhaustruct // not needed
		Status:    StatusStopped,
		ExitCode:  exitCode,
		StoppedAt: time.Now(),
	}
}

type ProcID uint64

func (id ProcID) String() string {
	return strconv.FormatUint(uint64(id), 10) //nolint:gomnd // decimal id
}

type ProcData struct {
	// Command - executable to run
	Command string
	Cwd     string
	Name    string
	// Args - arguments for executable, not including executable itself as first argument
	Args   []string
	Tags   []string
	Watch  []string
	Status Status
	ProcID ProcID

	// StdoutFile  string
	// StderrFile  string
	// RestartTries int
	// RestartDelay    time.Duration
	// Pid      int
	// Respawns int
}
