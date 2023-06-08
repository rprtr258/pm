package db

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rprtr258/simpdb"
	"github.com/rprtr258/xerr"
)

type StatusType int

const (
	StatusInvalid StatusType = iota
	StatusStarting
	StatusRunning
	StatusStopped
	StatusErrored
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
	case StatusErrored:
		return "errored"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", ps)
	}
}

type Status struct {
	StartTime time.Time  `json:"start_time"`
	Status    StatusType `json:"type"`
	// nulls if not running
	Pid int `json:"pid"`
	// CPU usage percentage rounded to integer
	CPU uint64 `json:"cpu"`
	// Memory usage in bytes
	Memory uint64 `json:"memory"`
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
	return Status{
		Status:    StatusRunning,
		StartTime: startTime,
		Pid:       pid,
		CPU:       cpu,
		Memory:    memory,
	}
}

func NewStatusStopped(exitCode int) Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusStopped,
		// TODO: add exit code
	}
}

func NewStatusErrored() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusErrored,
	}
}

func NewStatus() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusInvalid,
	}
}

type ProcID uint64

// TODO: implement String()
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
	return strconv.FormatUint(uint64(p.ProcID), 10)
}

type DBHandle struct {
	db    *simpdb.DB
	procs *simpdb.Table[ProcData]
}

// TODO: must call db.Close after this
func New(dir string) (DBHandle, error) {
	db := simpdb.New(dir)
	procs, err := simpdb.GetTable[ProcData](db, "procs", simpdb.TableConfig{
		Indent: false,
	})
	if err != nil {
		return DBHandle{}, err
	}

	return DBHandle{
		db:    db,
		procs: procs,
	}, nil
}

func (handle DBHandle) Close() error {
	return handle.procs.Flush()
}

func (handle DBHandle) AddProc(metadata ProcData) (ProcID, error) {
	maxProcID := uint64(0)
	handle.procs.Iter(func(id string, _ ProcData) bool {
		procID, _ := strconv.ParseUint(id, 10, 64) // TODO: remove, change ids to ints
		if procID > maxProcID {
			maxProcID = procID
		}

		return true
	})

	// TODO: remove mutation?
	metadata.ProcID = ProcID(maxProcID + 1)

	if !handle.procs.Insert(metadata) {
		return 0, errors.New("insert: already present")
	}

	if err := handle.procs.Flush(); err != nil {
		return 0, xerr.NewWM(err, "db flush")
	}

	return metadata.ProcID, nil
}

func (handle DBHandle) UpdateProc(metadata ProcData) error {
	handle.procs.Upsert(metadata)

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "db flush")
	}

	return nil
}

func (handle DBHandle) GetProcs(ids []ProcID) ([]ProcData, error) {
	lookupTable := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		lookupTable[strconv.FormatUint(uint64(id), 10)] = struct{}{}
	}

	res := handle.procs.Where(func(id string, _ ProcData) bool {
		_, ok := lookupTable[id]
		return ok
	}).List().All()

	return res, nil
}

func (handle DBHandle) List() map[ProcID]ProcData {
	res := map[ProcID]ProcData{}
	handle.procs.Iter(func(id string, pd ProcData) bool {
		res[pd.ProcID] = pd
		return true
	})

	return res
}

type ErrorProcNotFound ProcID

func (err ErrorProcNotFound) Error() string {
	return fmt.Sprintf("proc #%d not found", err)
}

func (handle DBHandle) SetStatus(procID ProcID, newStatus Status) error {
	pd := handle.procs.Get(strconv.FormatUint(uint64(procID), 10))
	if !pd.Valid {
		return ErrorProcNotFound(procID)
	}

	pd.Value.Status = newStatus
	handle.procs.Upsert(pd.Value)

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "db flush")
	}

	return nil
}

func (handle DBHandle) Delete(procIDs []uint64) error {
	lookupTable := make(map[uint64]struct{}, len(procIDs))
	for _, procID := range procIDs {
		lookupTable[procID] = struct{}{}
	}

	handle.procs.Where(func(_ string, pd ProcData) bool {
		_, ok := lookupTable[uint64(pd.ProcID)]
		return ok
	}).Delete()

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "db flush")
	}

	return nil
}
