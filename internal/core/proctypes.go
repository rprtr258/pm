package core

import (
	"bytes"
	rand2 "crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"text/template"
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
	Status StatusType

	// running
	StartTime time.Time // StartTime
	CPU       uint64    // CPU usage percentage rounded to integer
	Memory    uint64    // Memory usage in bytes

	// stopped
	ExitCode int
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

func NewStatusStopped(exitCode int) Status {
	return Status{ //nolint:exhaustruct // not needed
		Status:   StatusStopped,
		ExitCode: exitCode,
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

	Startup bool // Startup - run on OS startup

	// RestartTries int
	// RestartDelay    time.Duration
	// Respawns int
}

var _procStringTemplate = template.Must(template.New("proc").
	Parse(`Proc[
	id={{.ID}},
	command={{.Command}},
	cwd={{.Cwd}},
	name={{.Name}},
	args={{.Args}},
	tags={{.Tags}},
	watch={{if .Watch.Valid}}Some({{.Watch.Value}}){{else}}None{{end}},
	status={{.Status}},
	stdout_file={{.StdoutFile}},
	stderr_file={{.StderrFile}},
	startup={{.Startup}},
]`))

// TODO: add to template above
// "restart_tries": proc.RestartTries,
// "restart_delay": proc.RestartDelay,
// "respawns":     proc.Respawns,

func (p *Proc) String() string {
	var b bytes.Buffer
	_ = _procStringTemplate.Execute(&b, p)
	return b.String()
}
