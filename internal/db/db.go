package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/samber/lo"
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
	Cpu       uint64    `json:"cpu"`    // round(cpu usage in % * 100)
	Memory    uint64    `json:"memory"` // in bytes
}

type ProcID uint64

type DB map[ProcID]ProcData

// TODO: implement String()
type ProcData struct {
	ID   ProcID `json:"id"`
	Name string `json:"name"`
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

type DBHandle struct {
	dbFilename string
}

func encodeJSON[T any](proc T) ([]byte, error) {
	return json.Marshal(proc)
}

func decodeJSON[T any](data []byte) (T, error) {
	var res T
	if err := json.Unmarshal(data, &res); err != nil {
		return lo.Empty[T](), fmt.Errorf("failed decoding: %w", err)
	}

	return res, nil
}

// TODO: serialize/deserialize protobuffers

// TODO: must call db.Close after this
func New(dbFile string) DBHandle {
	return DBHandle{
		dbFilename: dbFile,
	}
}

func (handle DBHandle) Update(statement func(DB) error) error {
	db, err := os.ReadFile(handle.dbFilename)
	if err != nil {
		return fmt.Errorf("os.ReadFile failed: %w", err)
	}

	res, err := decodeJSON[DB](db)
	if err != nil {
		return fmt.Errorf("can't decode db file: %w", err)
	}

	if err := statement(res); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	toWrite, err := encodeJSON(res)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if err := os.WriteFile(handle.dbFilename, toWrite, 0o640); err != nil {
		return fmt.Errorf("database write failed: %w", err)
	}

	return nil
}

func (handle DBHandle) View(statement func(DB) error) error {
	db, err := os.ReadFile(handle.dbFilename)
	if err != nil {
		return fmt.Errorf("os.ReadFile failed: %w", err)
	}

	res, err := decodeJSON[DB](db)
	if err != nil {
		return fmt.Errorf("can't decode db file: %w", err)
	}

	return statement(res)
}

func (handle DBHandle) AddProc(metadata ProcData) (ProcID, error) {
	var newProcID uint64

	err := handle.Update(func(db DB) error {
		newProcID := ProcID(0)
		for {
			if _, ok := db[newProcID]; !ok {
				break
			}
			newProcID++
		}

		// TODO: remove mutation?
		metadata.ID = ProcID(newProcID)
		db[newProcID] = metadata

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("db.AddProc failed: %w", err)
	}

	return ProcID(newProcID), nil
}

func (handle DBHandle) GetProcs(ids []ProcID) ([]ProcData, error) {
	res := make([]ProcData, 0, len(ids))

	if err := handle.View(func(db DB) error {
		for _, procID := range ids {
			proc, ok := db[procID]
			if !ok {
				log.Printf("[WARN] can't find proc by id %d\n", procID)
			}

			res = append(res, proc)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("db.GetProcs failed: %w", err)
	}

	return res, nil
}

func (handle DBHandle) List() (map[ProcID]ProcData, error) {
	res := map[ProcID]ProcData{}

	if err := handle.View(func(db DB) error {
		res = db
		return nil
	}); err != nil {
		return nil, fmt.Errorf("db.List failed: %w", err)
	}

	return res, nil
}

func (handle DBHandle) SetStatus(procID ProcID, newStatus Status) error {
	return handle.Update(func(db DB) error {
		procData, ok := db[procID]
		if !ok {
			return fmt.Errorf("procID=%d not found", procID)
		}

		// TODO: remove mutation?
		procData.Status = newStatus
		db[procID] = procData

		return nil
	})
}

func (handle DBHandle) Delete(procIDs []uint64) error {
	return handle.Update(func(db DB) error {
		for _, procID := range procIDs {
			if _, ok := db[ProcID(procID)]; !ok {
				log.Printf("[WARN] procID=%d not found\n", procID)
			}

			delete(db, ProcID(procID))
		}

		return nil
	})
}

func (handle DBHandle) Init() error {
	_, err := os.Stat(handle.dbFilename)
	if err == nil {
		return nil
	} else if err != os.ErrNotExist {
		return fmt.Errorf("os.Stat failed: %w", err)
	}

	file, err := os.Create(handle.dbFilename)
	if err != nil {
		return fmt.Errorf("creating db failed: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString("{}"); err != nil {
		return fmt.Errorf("seeding db failed: %w", err)
	}

	return nil
}

// MapErr - like lo.Map but returns first error occured
func MapErr[T, R any](collection []T, iteratee func(T, int) (R, error)) ([]R, error) {
	results := make([]R, len(collection))
	for i, item := range collection {
		res, err := iteratee(item, i)
		if err != nil {
			return nil, fmt.Errorf("MapErr on i=%d: %w", i, err)
		}
		results[i] = res
	}
	return results, nil
}
