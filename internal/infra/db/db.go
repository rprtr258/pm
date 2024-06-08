package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	"github.com/rprtr258/pm/internal/core"
)

// status - db representation of core.Status
type status struct {
	StartTime time.Time `json:"start_time"` // StartTime, valid if running
	Status    int       `json:"type"`
	ExitCode  int       `json:"exit_code,omitempty"`
}

// procData - db representation of core.ProcData
type procData struct {
	ProcID core.PMID `json:"id"`
	Name   string    `json:"name"`
	Tags   []string  `json:"tags"`

	// Command - executable to run
	Command string `json:"command"`
	// Args - arguments for executable,
	// not including executable itself as first argument
	Args []string `json:"args"`
	// Cwd - working directory, should be absolute
	Cwd        string            `json:"cwd"`
	Env        map[string]string `json:"env"`
	StdoutFile string            `json:"stdout_file"`
	StderrFile string            `json:"stderr_file"`

	Watch  *string `json:"watch"`
	Status status  `json:"status"`

	Startup bool `json:"startup"`

	// RestartTries int
	// RestartDelay    time.Duration
	// Respawns int
}

func (p procData) ID() string {
	return p.ProcID.String()
}

func mapFromRepo(proc procData) core.Proc {
	return core.Proc{
		ID:      proc.ProcID,
		Command: proc.Command,
		Cwd:     proc.Cwd,
		Name:    proc.Name,
		Args:    proc.Args,
		Tags:    proc.Tags,
		Watch:   fun.FromPtr(proc.Watch),
		Status: core.Status{
			StartTime: proc.Status.StartTime,
			Status:    core.StatusType(proc.Status.Status),
			CPU:       0,
			Memory:    0,
			ExitCode:  proc.Status.ExitCode,
		},
		Env:        proc.Env,
		StdoutFile: proc.StdoutFile,
		StderrFile: proc.StderrFile,
		Startup:    proc.Startup,
	}
}

func InitRealDir(dir string) (afero.Fs, error) {
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("check directory %q: %w", dir, err)
		}

		if err := os.Mkdir(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create directory %q: %w", dir, err)
		}
	}

	return afero.NewBasePathFs(afero.NewOsFs(), dir), nil
}

type Handle struct {
	dir afero.Fs
}

func New(dir afero.Fs) Handle {
	return Handle{
		dir: dir,
	}
}

type CreateQuery struct {
	Name string   // Name of the process
	Tags []string // Tags - process tags

	Command    string            // Command - executable to run
	Args       []string          // Args - arguments for executable, not including executable itself as first argument
	Cwd        string            // Cwd - working directory
	Env        map[string]string // Env - environment variables
	StdoutFile fun.Option[string]
	StderrFile fun.Option[string]

	Watch fun.Option[string] // Watch - regex pattern for file watching

	Startup bool // Startup - should process be started on startup

	// RestartTries int
	// RestartDelay    time.Duration
	// Respawns int
}

func (h Handle) writeProc(proc procData) error {
	f, err := h.dir.OpenFile(proc.ProcID.String(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(proc)
}

func (h Handle) readProc(id core.PMID) (procData, error) {
	f, err := h.dir.Open(id.String())
	if err != nil {
		return procData{}, err
	}
	defer f.Close()

	var proc procData
	if err := json.NewDecoder(f).Decode(&proc); err != nil {
		return procData{}, err
	}

	return proc, nil
}

func (h Handle) AddProc(query CreateQuery, logsDir string) (core.PMID, error) {
	id := core.GenPMID()

	if err := h.writeProc(procData{
		ProcID:  id,
		Command: query.Command,
		Cwd:     query.Cwd,
		Name:    query.Name,
		Args:    query.Args,
		Tags:    query.Tags,
		Watch:   query.Watch.Ptr(),
		Status: status{ //nolint:exhaustruct // not needed
			Status: int(core.StatusCreated),
		},
		Env: query.Env,
		StdoutFile: query.StdoutFile.
			OrDefault(filepath.Join(logsDir, fmt.Sprintf("%s.stdout", id))),
		StderrFile: query.StderrFile.
			OrDefault(filepath.Join(logsDir, fmt.Sprintf("%s.stderr", id))),
		Startup: query.Startup,
	}); err != nil {
		return "", err
	}

	return id, nil
}

func (h Handle) UpdateProc(proc core.Proc) Error {
	if err := h.writeProc(procData{
		ProcID: proc.ID,
		Status: status{
			StartTime: proc.Status.StartTime,
			Status:    int(proc.Status.Status),
			ExitCode:  proc.Status.ExitCode,
		},
		Command:    proc.Command,
		Cwd:        proc.Cwd,
		Name:       proc.Name,
		Args:       proc.Args,
		Tags:       proc.Tags,
		Watch:      proc.Watch.Ptr(),
		Env:        proc.Env,
		StdoutFile: proc.StdoutFile,
		StderrFile: proc.StderrFile,
		Startup:    proc.Startup,
	}); err != nil {
		return FlushError{err}
	}

	return nil
}

func (h Handle) GetProc(id core.PMID) (core.Proc, bool) {
	proc, err := h.readProc(id)
	if err != nil {
		return fun.Zero[core.Proc](), false
	}

	return mapFromRepo(proc), true
}

func (h Handle) GetProcs(filterOpts ...core.FilterOption) (map[core.PMID]core.Proc, error) {
	entries, err := afero.ReadDir(h.dir, ".")
	if err != nil {
		return nil, err
	}

	procs := map[core.PMID]core.Proc{}
	for _, entry := range entries {
		proc, err := h.readProc(core.PMID(entry.Name()))
		if err != nil {
			return nil, err
		}

		procs[proc.ProcID] = mapFromRepo(proc)
	}

	return fun.SliceToMap[core.PMID, core.Proc](
		func(id core.PMID) (core.PMID, core.Proc) {
			return id, procs[id]
		},
		core.FilterProcMap(procs, filterOpts...)...), nil
}

func (h Handle) StatusSet(id core.PMID, newStatus core.Status) Error {
	proc, err := h.readProc(id)
	if err != nil {
		return ProcNotFoundError{id}
	}

	proc.Status = status{
		StartTime: newStatus.StartTime,
		Status:    int(newStatus.Status),
		ExitCode:  newStatus.ExitCode,
	}

	if err := h.writeProc(proc); err != nil {
		return FlushError{err}
	}

	return nil
}

func (h Handle) StatusSetSafe(id core.PMID, newStatus core.Status) {
	if err := h.StatusSet(id, newStatus); err != nil {
		log.Error().
			Stringer("pmid", id).
			Any("new_status", newStatus).
			Msg("set proc status to running")
	}
}

func (h Handle) Delete(id core.PMID) (core.Proc, Error) {
	proc, err := h.readProc(id)
	if err != nil {
		return fun.Zero[core.Proc](), ProcNotFoundError{id}
	}

	if err := h.dir.Remove(id.String()); err != nil {
		return fun.Zero[core.Proc](), FlushError{err}
	}

	return mapFromRepo(proc), nil
}
