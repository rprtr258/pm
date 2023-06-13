package core

import (
	"fmt"
	"strconv"
	"time"
)

type StatusType int

const (
	StatusInvalid StatusType = iota
	StatusStarting
	StatusRunning
	StatusStopped
)

func (ps StatusType) String() string {
	switch ps {
	case StatusInvalid:
		return "invalid"
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusStopped:
		return "stopped"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", ps)
	}
}

type Status struct {
	StartTime time.Time  `json:"start_time"` // StartTime, valid if running
	StoppedAt time.Time  `json:"stopped_at"` // StoppedAt - time when the process stopped, valid if stopped
	Status    StatusType `json:"type"`
	Pid       int        `json:"pid"`       // PID, valid if running
	CPU       uint64     `json:"cpu"`       // CPU usage percentage rounded to integer, valid if running
	Memory    uint64     `json:"memory"`    // Memory usage in bytes, valid if running
	ExitCode  int        `json:"exit_code"` // ExitCode of the process, valid if stopped
}

func NewStatusInvalid() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusInvalid,
	}
}

func NewStatusStarting() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusStarting,
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

func NewStatus() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusInvalid,
	}
}

type ProcID uint64

func (id ProcID) String() string {
	return strconv.FormatUint(uint64(id), 10) //nolint:gomnd // decimal id
}

type ProcData struct {
	// Command - executable to run
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
	Name    string `json:"name"`
	// Args - arguments for executable, not including executable itself as first argument
	Args   []string `json:"args"`
	Tags   []string `json:"tags"`
	Watch  []string `json:"watch"`
	Status Status   `json:"status"`
	ProcID ProcID   `json:"id"`

	// StdoutFile  string
	// StderrFile  string
	// RestartTries int
	// RestartDelay    time.Duration
	// Pid      int
	// Respawns int
}

func (p ProcData) ID() string {
	return p.ProcID.String()
}
