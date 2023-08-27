package db

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/simpdb"
	"github.com/rprtr258/simpdb/storages"

	"github.com/rprtr258/pm/internal/core"
)

// status - db representation of core.Status
type status struct {
	StartTime time.Time `json:"start_time"` // StartTime, valid if running
	StoppedAt time.Time `json:"stopped_at"` // StoppedAt - time when the process stopped, valid if stopped
	Status    int       `json:"type"`
	Pid       int       `json:"pid"`       // PID, valid if running
	ExitCode  int       `json:"exit_code"` // ExitCode of the process, valid if stopped
}

// procData - db representation of core.ProcData
type procData struct {
	ProcID core.ProcID `json:"id"`
	Name   string      `json:"name"`
	Tags   []string    `json:"tags"`

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

	// RestartTries int
	// RestartDelay    time.Duration
	// Respawns int
}

func (p procData) ID() string {
	return strconv.FormatUint(p.ProcID, 10)
}

type Handle struct {
	db    *simpdb.DB
	procs *simpdb.Table[procData]
}

func New(dir string) (Handle, Error) {
	db := simpdb.New(dir)

	procs, errTableProcs := simpdb.GetTable(db, "procs", storages.NewJSONStorage[procData]())
	if errTableProcs != nil {
		return fun.Zero[Handle](), GetTableError{"procs"}
	}

	return Handle{
		db:    db,
		procs: procs,
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

func (handle Handle) AddProc(query CreateQuery, logsDir string) (core.ProcID, Error) {
	maxProcID := core.ProcID(0)
	handle.procs.Iter(func(_ string, proc procData) bool {
		maxProcID = max(maxProcID, proc.ProcID)
		return true
	})

	newProcID := maxProcID + 1

	handle.procs.Insert(procData{
		ProcID:  newProcID,
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
			OrDefault(filepath.Join(logsDir, fmt.Sprintf("%d.stdout", newProcID))),
		StderrFile: query.StderrFile.
			OrDefault(filepath.Join(logsDir, fmt.Sprintf("%d.stderr", newProcID))),
	})

	if err := handle.procs.Flush(); err != nil {
		return 0, FlushError{err}
	}

	return newProcID, nil
}

func (handle Handle) UpdateProc(proc core.Proc) Error {
	handle.procs.Upsert(procData{
		ProcID: proc.ID,
		Status: status{
			StartTime: proc.Status.StartTime,
			StoppedAt: proc.Status.StoppedAt,
			Status:    int(proc.Status.Status),
			Pid:       proc.Status.Pid,
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
	})

	if err := handle.procs.Flush(); err != nil {
		return FlushError{err}
	}

	return nil
}

func (handle Handle) GetProc(id core.ProcID) (core.Proc, bool) {
	procs := handle.GetProcs(core.WithIDs(id))
	if len(procs) != 1 {
		return fun.Zero[core.Proc](), false
	}

	return procs[id], true
}

func (handle Handle) GetProcs(filterOpts ...core.FilterOption) core.Procs {
	filter := core.NewFilter(filterOpts...)

	procs := core.Procs{}
	handle.procs.
		Iter(func(id string, proc procData) bool {
			procs[proc.ProcID] = core.Proc{
				ID:      proc.ProcID,
				Command: proc.Command,
				Cwd:     proc.Cwd,
				Name:    proc.Name,
				Args:    proc.Args,
				Tags:    proc.Tags,
				Watch:   fun.FromPtr(proc.Watch),
				Status: core.Status{
					StartTime: proc.Status.StartTime,
					StoppedAt: proc.Status.StoppedAt,
					Status:    core.StatusType(proc.Status.Status),
					Pid:       proc.Status.Pid,
					ExitCode:  proc.Status.ExitCode,
					CPU:       0,
					Memory:    0,
				},
				Env:        proc.Env,
				StdoutFile: proc.StdoutFile,
				StderrFile: proc.StderrFile,
			}

			return true
		})

	return fun.SliceToMap[core.ProcID, core.Proc](
		core.FilterProcMap(procs, filter),
		func(id core.ProcID) (core.ProcID, core.Proc) {
			return id, procs[id]
		})
}

func (handle Handle) SetStatus(procID core.ProcID, newStatus core.Status) Error {
	proc, ok := handle.procs.Get(strconv.FormatUint(procID, 10))
	if !ok {
		return ProcNotFoundError{procID}
	}

	proc.Status = status{
		StartTime: newStatus.StartTime,
		StoppedAt: newStatus.StoppedAt,
		Status:    int(newStatus.Status),
		Pid:       newStatus.Pid,
		ExitCode:  newStatus.ExitCode,
	}
	handle.procs.Upsert(proc)

	if err := handle.procs.Flush(); err != nil {
		return FlushError{err}
	}

	return nil
}

func (handle Handle) Delete(procID core.ProcID) (core.Proc, Error) {
	deletedProcs := handle.procs.
		Where(func(_ string, proc procData) bool {
			return proc.ProcID == procID
		}).
		Delete()

	if err := handle.procs.Flush(); err != nil {
		return fun.Zero[core.Proc](), FlushError{err}
	}

	if len(deletedProcs) == 0 {
		return fun.Zero[core.Proc](), ProcNotFoundError{procID}
	}

	proc := deletedProcs[0]
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
			StoppedAt: proc.Status.StoppedAt,
			Status:    core.StatusType(proc.Status.Status),
			Pid:       proc.Status.Pid,
			ExitCode:  proc.Status.ExitCode,
			CPU:       0,
			Memory:    0,
		},
		Env:        proc.Env,
		StdoutFile: proc.StdoutFile,
		StderrFile: proc.StderrFile,
	}, nil
}
