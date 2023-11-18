package core

import (
	rand2 "crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
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

// PMID is a unique identifier for a process
type PMID string

func (pmid PMID) String() string {
	return string(pmid)
}

func GenPMID() PMID {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand2.Reader, b); err != nil {
		// fallback to random string
		for i := range b {
			b[i] = byte(rand.Intn(256)) //nolint:gosec // fuck you
		}
	}

	return PMID(hex.EncodeToString(b))
}

type Status struct {
	StartTime time.Time // StartTime, valid if running
	Status    StatusType
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

func NewStatusRunning(startTime time.Time, cpu, memory uint64) Status {
	return Status{ //nolint:exhaustruct // not needed
		Status:    StatusRunning,
		StartTime: startTime,
		CPU:       cpu,
		Memory:    memory,
	}
}

func NewStatusStopped() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusStopped,
	}
}

type Proc struct {
	ID   PMID
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

type Procs = map[PMID]Proc
