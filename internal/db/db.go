package db

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rprtr258/simpdb"
)

type ProcStatus int

const (
	StatusInvalid ProcStatus = iota
	StatusStarting
	StatusRunning
	StatusStopped
	StatusErrored
)

type Status struct {
	Status ProcStatus `json:"status"`
	// nulls if not running
	Pid       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
	// Cpu usage percentage rounded to integer
	Cpu uint64 `json:"cpu"`
	// Memory usage in bytes
	Memory uint64 `json:"memory"`
}

type ProcID uint64

// TODO: implement String()
type ProcData struct {
	ProcID ProcID `json:"id"`
	Name   string `json:"name"`
	// Command - executable to run
	Command string `json:"command"`
	// Args - arguments for executable, not including executable itself as first argument
	Args   []string `json:"args"`
	Status Status   `json:"status"`
	Tags   []string `json:"tags"`
	Cwd    string   `json:"cwd"`
	Watch  []string `json:"watch"`
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
func New(dbFile string) (DBHandle, error) {
	db := simpdb.New(dbFile) // TODO: dbDir
	procs, err := simpdb.GetTable[ProcData](db, "procs", simpdb.TableConfig{})
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

	return metadata.ProcID, nil
}

func (handle DBHandle) UpdateProc(metadata ProcData) {
	handle.procs.Upsert(metadata)
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

func (handle DBHandle) SetStatus(procID ProcID, newStatus Status) error {
	pd := handle.procs.Get(strconv.FormatUint(uint64(procID), 10))
	if !pd.Valid {
		return fmt.Errorf("proc %d was not found", procID)
	}

	pd.Value.Status = newStatus
	handle.procs.Upsert(pd.Value)
	return nil
}

func (handle DBHandle) Delete(procIDs []uint64) {
	lookupTable := make(map[uint64]struct{}, len(procIDs))
	for _, procID := range procIDs {
		lookupTable[procID] = struct{}{}
	}

	handle.procs.Where(func(_ string, pd ProcData) bool {
		_, ok := lookupTable[uint64(pd.ProcID)]
		return ok
	}).Delete()
}
