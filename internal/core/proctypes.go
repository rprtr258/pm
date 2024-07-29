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

const PMIDLen = 32

// PMID is a unique identifier for a process
type PMID string

func (pmid PMID) String() string {
	return string(pmid)
}

func GenPMID() PMID {
	// one byte is two hex digits, so divide by two
	b := make([]byte, PMIDLen/2)
	if _, err := io.ReadFull(rand2.Reader, b); err != nil {
		// fallback to random string
		for i := range b {
			b[i] = byte(rand.Intn(256)) //nolint:gosec // fuck you
		}
	}

	return PMID(hex.EncodeToString(b))
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

	Watch fun.Option[string]

	Startup bool // Startup - run on OS startup

	KillTimeout time.Duration // time to wait before sending SIGKILL
	DependsOn   []string      // names of processes that must be started before this proc
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

func (p *Proc) String() string {
	var b bytes.Buffer
	_ = _procStringTemplate.Execute(&b, p)
	return b.String()
}

type Status int

const (
	StatusCreated Status = iota
	StatusRunning
	StatusStopped
)

func (ps Status) String() string {
	switch ps {
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

// ProcStat is Proc with Stat!
// Stat means current status.
type ProcStat struct {
	Proc
	Status    Status
	StartTime time.Time
	CPU       float64
	Memory    uint64
	ShimPID   int
	ChildPID  fun.Option[int]
}
