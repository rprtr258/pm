package db

import (
	"fmt"
	"time"

	"github.com/rprtr258/simpdb"
	"github.com/rprtr258/simpdb/storages"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core"
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

func NewStatusRunning(startTime time.Time, pid int) Status {
	return Status{ //nolint:exhaustruct // not needed
		Status:    StatusRunning,
		StartTime: startTime,
		Pid:       pid,
	}
}

func NewStatusStopped(exitCode int) Status {
	return Status{ //nolint:exhaustruct // not needed
		Status:    StatusStopped,
		ExitCode:  exitCode,
		StoppedAt: time.Now(),
	}
}

type ProcData struct {
	// Command - executable to run
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
	Name    string `json:"name"`
	// Args - arguments for executable, not including executable itself as first argument
	Args   []string    `json:"args"`
	Tags   []string    `json:"tags"`
	Watch  []string    `json:"watch"`
	Status Status      `json:"status"`
	ProcID core.ProcID `json:"id"`

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

type Handle struct {
	db    *simpdb.DB
	procs *simpdb.Table[ProcData]
}

func New(dir string) (Handle, error) {
	db := simpdb.New(dir)

	procs, err := simpdb.GetTable(db, "procs", storages.NewJSONStorage[ProcData]())
	if err != nil {
		return Handle{}, err
	}

	return Handle{
		db:    db,
		procs: procs,
	}, nil
}

type CreateQuery struct {
	// Command - executable to run
	Command string
	Cwd     string
	Name    string
	// Args - arguments for executable, not including executable itself as first argument
	Args  []string
	Tags  []string
	Watch []string

	// StdoutFile  string
	// StderrFile  string
	// RestartTries int
	// RestartDelay    time.Duration
	// Pid      int
	// Respawns int
}

func (handle Handle) AddProc(query CreateQuery) (core.ProcID, error) {
	maxProcID := core.ProcID(0)
	handle.procs.Iter(func(_ string, proc ProcData) bool {
		if proc.ProcID > maxProcID {
			maxProcID = proc.ProcID
		}

		return true
	})

	newProcID := maxProcID + 1

	if !handle.procs.Insert(ProcData{
		ProcID:  newProcID,
		Command: query.Command,
		Cwd:     query.Cwd,
		Name:    query.Name,
		Args:    query.Args,
		Tags:    query.Tags,
		Watch:   query.Watch,
		Status:  NewStatusStarting(),
	}) {
		return 0, xerr.NewM("insert: already present")
	}

	if err := handle.procs.Flush(); err != nil {
		return 0, xerr.NewWM(err, "insert: db flush")
	}

	return newProcID, nil
}

func (handle Handle) UpdateProc(metadata ProcData) error {
	handle.procs.Upsert(metadata)

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "update: db flush")
	}

	return nil
}

func (handle Handle) GetProcs(ids []core.ProcID) []ProcData {
	lookupTable := lo.SliceToMap(ids, func(id core.ProcID) (string, struct{}) {
		return id.String(), struct{}{}
	})

	return handle.procs.
		Where(func(id string, _ ProcData) bool {
			_, ok := lookupTable[id]
			return ok
		}).
		List().
		All()
}

func (handle Handle) List() map[core.ProcID]ProcData {
	res := map[core.ProcID]ProcData{}
	handle.procs.Iter(func(id string, pd ProcData) bool {
		res[pd.ProcID] = pd
		return true
	})
	return res
}

type ProcNotFoundError core.ProcID

func (err ProcNotFoundError) Error() string {
	return fmt.Sprintf("proc #%d not found", err)
}

func (handle Handle) SetStatus(procID core.ProcID, newStatus Status) error {
	proc, ok := handle.procs.Get(procID.String())
	if !ok {
		return ProcNotFoundError(procID)
	}

	if newStatus.Status == StatusStopped {
		newStatus.StartTime = proc.Status.StartTime
	}

	proc.Status = newStatus
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

	handle.procs.Where(func(_ string, pd ProcData) bool {
		_, ok := lookupTable[pd.ProcID]
		return ok
	}).Delete()

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "delete: db flush")
	}

	return nil
}
