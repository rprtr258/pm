package core

import (
	"fmt"
	"time"

	"github.com/rprtr258/fun"
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
	Status    StatusType
	Pid       int    // PID, valid if running
	CPU       uint64 // CPU usage percentage rounded to integer, valid if running
	Memory    uint64 // Memory usage in bytes, valid if running
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

func NewStatusStopped() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusStopped,
	}
}

type ProcID = uint64

type Proc struct {
	ID   ProcID
	Name string
	Tags []string

	Command    string            // Command - executable to run
	Args       []string          // Args - arguments for executable, not including executable itself as first argument
	Cwd        string            // Cwd - working directory, must be absolute
	Env        map[string]string // Env - process environment
	StdoutFile string
	StderrFile string

	Watch  fun.Option[string]
	Status Status

	// RestartTries int
	// RestartDelay    time.Duration
	// Respawns int
}

type Procs = map[ProcID]Proc
