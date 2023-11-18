package db

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/simpdb"
	"github.com/rprtr258/simpdb/storages"

	"github.com/rprtr258/pm/internal/core"
)

// status - db representation of core.Status
type status struct {
	StartTime time.Time `json:"start_time"` // StartTime, valid if running
	Status    int       `json:"type"`
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

	// RestartTries int
	// RestartDelay    time.Duration
	// Respawns int
}

func (p procData) ID() string {
	return p.ProcID.String()
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

func (handle Handle) AddProc(query CreateQuery, logsDir string) (core.PMID, Error) {
	id := core.GenPMID()
	handle.procs.Insert(procData{
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
			OrDefault(filepath.Join(logsDir, fmt.Sprintf("%d.stdout", id))),
		StderrFile: query.StderrFile.
			OrDefault(filepath.Join(logsDir, fmt.Sprintf("%d.stderr", id))),
	})

	if err := handle.procs.Flush(); err != nil {
		return "", FlushError{err}
	}

	return id, nil
}

func (handle Handle) UpdateProc(proc core.Proc) Error {
	handle.procs.Upsert(procData{
		ProcID: proc.ID,
		Status: status{
			StartTime: proc.Status.StartTime,
			Status:    int(proc.Status.Status),
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

func (handle Handle) GetProc(id core.PMID) (core.Proc, bool) {
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
					Status:    core.StatusType(proc.Status.Status),
					CPU:       0,
					Memory:    0,
				},
				Env:        proc.Env,
				StdoutFile: proc.StdoutFile,
				StderrFile: proc.StderrFile,
			}

			return true
		})

	return fun.SliceToMap[core.PMID, core.Proc](
		core.FilterProcMap(procs, filter),
		func(id core.PMID) (core.PMID, core.Proc) {
			return id, procs[id]
		})
}

func (handle Handle) SetStatus(id core.PMID, newStatus core.Status) Error {
	proc, ok := handle.procs.Get(id.String())
	if !ok {
		return ProcNotFoundError{id}
	}

	proc.Status = status{
		StartTime: newStatus.StartTime,
		Status:    int(newStatus.Status),
	}
	handle.procs.Upsert(proc)

	if err := handle.procs.Flush(); err != nil {
		return FlushError{err}
	}

	return nil
}

func (handle Handle) Delete(id core.PMID) (core.Proc, Error) {
	deletedProcs := handle.procs.
		Where(func(_ string, proc procData) bool {
			return proc.ProcID == id
		}).
		Delete()

	if err := handle.procs.Flush(); err != nil {
		return fun.Zero[core.Proc](), FlushError{err}
	}

	if len(deletedProcs) == 0 {
		return fun.Zero[core.Proc](), ProcNotFoundError{id}
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
			Status:    core.StatusType(proc.Status.Status),
			CPU:       0,
			Memory:    0,
		},
		Env:        proc.Env,
		StdoutFile: proc.StdoutFile,
		StderrFile: proc.StderrFile,
	}, nil
}
