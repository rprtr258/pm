package db

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/simpdb"
	"github.com/rprtr258/simpdb/storages"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"

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
	// Command - executable to run
	Command string `json:"command"`
	// Cwd - working directory, should be absolute
	Cwd  string `json:"cwd"`
	Name string `json:"name"`
	// Args - arguments for executable,
	// not including executable itself as first argument
	Args       []string          `json:"args"`
	Tags       []string          `json:"tags"`
	Watch      *string           `json:"watch"`
	Status     status            `json:"status"`
	ProcID     core.ProcID       `json:"id"`
	Env        map[string]string `json:"env"`
	StdoutFile string            `json:"stdout_file"`
	StderrFile string            `json:"stderr_file"`

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

func New(dir string) (Handle, error) {
	db := simpdb.New(dir)

	procs, errTableProcs := simpdb.GetTable(db, "procs", storages.NewJSONStorage[procData]())
	if errTableProcs != nil {
		return Handle{}, xerr.NewWM(errTableProcs, "get table", xerr.Fields{"table": "procs"})
	}

	return Handle{
		db:    db,
		procs: procs,
	}, nil
}

type CreateQuery struct {
	// Env - environment variables
	Env map[string]string
	// Command - executable to run
	Command string
	// Cwd - working directory
	Cwd string
	// Name - name of the process
	Name string
	// Args - arguments for executable, not including executable itself as first argument
	Args []string
	// Tags - process tags
	Tags []string
	// Watch - regex pattern for file watching
	Watch      fun.Option[string]
	StdoutFile fun.Option[string]
	StderrFile fun.Option[string]

	// RestartTries int
	// RestartDelay    time.Duration
	// Respawns int
}

func (handle Handle) AddProc(query CreateQuery, logsDir string) (core.ProcID, error) {
	maxProcID := core.ProcID(0)
	handle.procs.Iter(func(_ string, proc procData) bool {
		if proc.ProcID > maxProcID {
			maxProcID = proc.ProcID
		}

		return true
	})

	newProcID := maxProcID + 1

	if !handle.procs.Insert(procData{
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
	}) {
		return 0, xerr.NewM("insert: already present")
	}

	if err := handle.procs.Flush(); err != nil {
		return 0, xerr.NewWM(err, "insert: db flush")
	}

	return newProcID, nil
}

func (handle Handle) UpdateProc(proc core.ProcData) error {
	handle.procs.Upsert(procData{
		ProcID: proc.ProcID,
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
		return xerr.NewWM(err, "update: db flush")
	}

	return nil
}

func (handle Handle) GetProc(id core.ProcID) (core.ProcData, bool) {
	procs := handle.GetProcs([]core.ProcID{id})
	if len(procs) != 1 {
		return fun.Zero[core.ProcData](), false
	}

	return procs[0], true
}

func (handle Handle) GetProcs(ids []core.ProcID) map[core.ProcID]core.ProcData {
	lookupTable := lo.SliceToMap(ids, func(id core.ProcID) (string, struct{}) {
		return id.String(), struct{}{}
	})

	res := make(map[core.ProcID]core.ProcData, len(lookupTable))

	handle.procs.
		Where(func(id string, _ procData) bool {
			_, ok := lookupTable[id]
			return ok
		}).
		Iter(func(id string, proc procData) bool {
			res[proc.ProcID] = core.ProcData{
				ProcID:  proc.ProcID,
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

	return res
}

// TODO: merge with GetProcs, use filters
func (handle Handle) List() map[core.ProcID]core.ProcData {
	res := map[core.ProcID]core.ProcData{}
	handle.procs.Iter(func(id string, proc procData) bool {
		res[proc.ProcID] = core.ProcData{
			ProcID:  proc.ProcID,
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
	return res
}

type ProcNotFoundError core.ProcID

func (err ProcNotFoundError) Error() string {
	return fmt.Sprintf("proc #%d not found", err)
}

func (handle Handle) SetStatus(procID core.ProcID, newStatus core.Status) error {
	proc, ok := handle.procs.Get(procID.String())
	if !ok {
		return ProcNotFoundError(procID)
	}

	// TODO: ???
	if newStatus.Status == core.StatusStopped {
		newStatus.StartTime = proc.Status.StartTime
	}

	proc.Status = status{
		StartTime: newStatus.StartTime,
		StoppedAt: proc.Status.StoppedAt,
		Status:    int(newStatus.Status),
		Pid:       proc.Status.Pid,
		ExitCode:  proc.Status.ExitCode,
	}
	handle.procs.Upsert(proc)

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "set status: db flush")
	}

	return nil
}

func (handle Handle) Delete(procIDs []uint64) error {
	lookupTable := lo.SliceToMap(procIDs, func(id uint64) (core.ProcID, struct{}) {
		return core.ProcID(id), struct{}{}
	})

	handle.procs.Where(func(_ string, proc procData) bool {
		_, ok := lookupTable[proc.ProcID]
		return ok
	}).Delete()

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "delete: db flush")
	}

	return nil
}
