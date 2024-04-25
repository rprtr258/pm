package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"

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

type Handle struct {
	dir string
}

func New(dir string) (Handle, error) {
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return Handle{}, fmt.Errorf("check directory: %w", err)
		}

		if err := os.Mkdir(dir, 0o755); err != nil {
			return Handle{}, fmt.Errorf("create directory: %w", err)
		}
	}

	return Handle{
		dir: dir,
	}, nil
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

	// RestartTries int
	// RestartDelay    time.Duration
	// Respawns int
}

func (handle Handle) writeProc(proc procData) error {
	f, err := os.OpenFile(filepath.Join(handle.dir, proc.ProcID.String()), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = json.NewEncoder(f).Encode(proc); err != nil {
		return err
	}

	return nil
}

func (handle Handle) readProc(id core.PMID) (procData, error) {
	f, err := os.Open(filepath.Join(handle.dir, id.String()))
	if err != nil {
		return procData{}, err
	}

	var proc procData
	if err := json.NewDecoder(f).Decode(&proc); err != nil {
		return procData{}, err
	}

	return proc, nil
}

func (handle Handle) AddProc(query CreateQuery, logsDir string) (core.PMID, error) {
	id := core.GenPMID()

	if err := handle.writeProc(procData{
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
	}); err != nil {
		return "", err
	}

	return id, nil
}

func (handle Handle) UpdateProc(proc core.Proc) Error {
	if err := handle.writeProc(procData{
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

func (handle Handle) GetProc(id core.PMID) (core.Proc, bool) {
	proc, err := handle.readProc(id)
	if err != nil {
		return fun.Zero[core.Proc](), false
	}

	return mapFromRepo(proc), true
}

func (handle Handle) GetProcs(filterOpts ...core.FilterOption) (core.Procs, error) {
	entries, err := os.ReadDir(handle.dir)
	if err != nil {
		return nil, err
	}

	procs := core.Procs{}
	for _, entry := range entries {
		proc, err := handle.readProc(core.PMID(entry.Name()))
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

func (handle Handle) SetStatus(id core.PMID, newStatus core.Status) Error {
	proc, err := handle.readProc(id)
	if err != nil {
		return ProcNotFoundError{id}
	}

	proc.Status = status{
		StartTime: newStatus.StartTime,
		Status:    int(newStatus.Status),
		ExitCode:  newStatus.ExitCode,
	}

	if err := handle.writeProc(proc); err != nil {
		return FlushError{err}
	}

	return nil
}

func (handle Handle) Delete(id core.PMID) (core.Proc, Error) {
	proc, err := handle.readProc(id)
	if err != nil {
		return fun.Zero[core.Proc](), ProcNotFoundError{id}
	}

	if err := os.Remove(filepath.Join(handle.dir, id.String())); err != nil {
		return fun.Zero[core.Proc](), FlushError{err}
	}

	return mapFromRepo(proc), nil
}

func (handle Handle) StatusSetRunning(id core.PMID) {
	// TODO: fill/remove cpu, memory
	runningStatus := core.NewStatusRunning(time.Now(), 0, 0)
	if err := handle.SetStatus(id, runningStatus); err != nil {
		log.Error().
			Stringer("pmid", id).
			Any("new_status", runningStatus).
			Msg("set proc status to running")
	}
}

func (handle Handle) StatusSetStopped(id core.PMID, exitCode int) {
	dbStatus := core.NewStatusStopped(exitCode)
	if err := handle.SetStatus(id, dbStatus); err != nil {
		if _, ok := err.(ProcNotFoundError); ok {
			log.Error().
				Stringer("pmid", id).
				Msg("proc not found while trying to set stopped status")
		} else {
			log.Error().
				Stringer("pmid", id).
				Any("new_status", dbStatus).
				Msg("set proc status to stopped")
		}
	}
}
