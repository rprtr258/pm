package db

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rprtr258/simpdb"
	"github.com/rprtr258/simpdb/storages"
	"github.com/rprtr258/xerr"
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
	CPU       uint64     `json:"cpu"`       // CPU usage percentage rounded to integer, valid if running
	Memory    uint64     `json:"memory"`    // Memory usage in bytes, valid if running
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

func NewStatusRunning(startTime time.Time, pid int, cpu, memory uint64) Status {
	return Status{ //nolint:exhaustruct // not needed
		Status:    StatusRunning,
		StartTime: startTime,
		Pid:       pid,
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

func NewStatus() Status {
	return Status{ //nolint:exhaustruct // not needed
		Status: StatusInvalid,
	}
}

type ProcID uint64

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
	return strconv.FormatUint(uint64(p.ProcID), 10) //nolint:gomnd // decimal id
}

type Handle struct {
	db    *simpdb.DB
	procs *simpdb.Table[ProcData]
}

// TODO: must call db.Close after this
func New(dir string) (Handle, error) {
	db := simpdb.New(dir)

	procs, err := simpdb.GetTable[ProcData](db, "procs", storages.NewJSONStorage[ProcData]())
	if err != nil {
		return Handle{}, err
	}

	return Handle{
		db:    db,
		procs: procs,
	}, nil
}

func (handle Handle) Close() error {
	if errFlush := handle.procs.Flush(); errFlush != nil {
		return xerr.NewWM(errFlush, "flush procs table")
	}

	return nil
}

func (handle Handle) AddProc(metadata ProcData) (ProcID, error) {
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

func (handle Handle) UpdateProc(metadata ProcData) error {
	handle.procs.Upsert(metadata)

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "db flush")
	}

	return nil
}

func (handle Handle) GetProcs(ids []ProcID) ([]ProcData, error) {
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

func (handle Handle) List() map[ProcID]ProcData {
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

func (handle Handle) SetStatus(procID ProcID, newStatus Status) error {
	procDataMaybe := handle.procs.Get(strconv.FormatUint(uint64(procID), 10)) //nolint:gomnd // decimal
	if !procDataMaybe.Valid {
		return ErrorProcNotFound(procID)
	}

	procDataMaybe.Value.Status = newStatus
	handle.procs.Upsert(procDataMaybe.Value)

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "db flush")
	}

	return nil
}

func (handle Handle) Delete(procIDs []uint64) error {
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
