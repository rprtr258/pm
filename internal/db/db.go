package db

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
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

func encodeUint64(procID uint64) []byte {
	keyBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(keyBytes, procID)
	return keyBytes
}

func encodeJSON[T any](proc T) ([]byte, error) {
	return json.Marshal(proc)
}

func decodeJSON[T any](item *badger.Item) (T, error) {
	var res T
	if err := item.Value(func(valueBuffer []byte) error {
		if err := json.Unmarshal(valueBuffer, &res); err != nil {
			return fmt.Errorf("failed decoding value=%v: %w", valueBuffer, err)
		}
		return nil
	}); err != nil {
		return lo.Empty[T](), err
	}

	return res, nil
}

// TODO: serialize/deserialize protobuffers
func putProcMetadata(tx *badger.Txn, procID uint64, value ProcData) error {
	keyBytes := encodeUint64(procID)

	valueBytes, err := encodeJSON(value)
	if err != nil {
		return err
	}

	if err := tx.Set(keyBytes, valueBytes); err != nil {
		return fmt.Errorf("tx.Set failed: %w", err)
	}

	return nil
}

// TODO: must call db.Close after this
func New(dbFile string) DBHandle {
	return DBHandle{
		dbFilename: dbFile,
	}
}

func (handle DBHandle) Update(statement func(*badger.Txn) error) error {
	// TODO: in dev,daemon change log level
	db, err := badger.Open(badger.DefaultOptions(handle.dbFilename).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return fmt.Errorf("bbolt.Open failed: %w", err)
	}
	defer db.Close()

	return db.Update(statement)
}

func (handle DBHandle) View(statement func(*badger.Txn) error) error {
	db, err := badger.Open(badger.DefaultOptions(handle.dbFilename).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return fmt.Errorf("bbolt.Open failed: %w", err)
	}
	defer db.Close()

	return db.View(statement)
}

func (handle DBHandle) AddProc(metadata ProcData) (ProcID, error) {
	var newProcID uint64

	err := handle.Update(func(tx *badger.Txn) error {
		procIDs := map[uint64]struct{}{}
		func() {
			it := tx.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()

			for it.Rewind(); it.Valid(); it.Next() {
				procID := binary.BigEndian.Uint64(it.Item().Key())
				procIDs[procID] = struct{}{}
			}
		}()

		newProcID = mex(procIDs)

		// TODO: remove mutation?
		metadata.ID = ProcID(newProcID)

		if err := putProcMetadata(tx, newProcID, metadata); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("db.AddProc failed: %w", err)
	}

	return ProcID(newProcID), nil
}

func (handle DBHandle) GetProcs(ids []ProcID) ([]ProcData, error) {
	var res []ProcData

	if err := handle.View(func(tx *badger.Txn) error {
		var err error
		res, err = MapErr(ids, func(procID ProcID, _ int) (ProcData, error) {
			keyBuffer := encodeUint64(uint64(procID))

			item, err := tx.Get(keyBuffer)
			if err != nil {
				return ProcData{}, fmt.Errorf("failed getting id=%d: %w", procID, err)
			}

			procData, err := decodeJSON[ProcData](item)
			if err != nil {
				return ProcData{}, fmt.Errorf("decoding procID=%d failed: %w", procID, err)
			}

			return procData, nil
		})
		if err != nil {
			return fmt.Errorf("scanning failed: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("db.GetProcs failed: %w", err)
	}

	return res, nil
}

func (handle DBHandle) List() ([]ProcData, error) {
	var res []ProcData

	if err := handle.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			procData, err := decodeJSON[ProcData](item)
			if err != nil {
				return err
			}

			res = append(res, procData)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("db.List failed: %w", err)
	}

	return res, nil
}

func (handle DBHandle) SetStatus(procID ProcID, newStatus Status) error {
	return handle.Update(func(tx *badger.Txn) error {
		key := encodeUint64(uint64(procID))

		item, err := tx.Get(key)
		if err != nil {
			return fmt.Errorf("reading procID=%v failed: %w", procID, err)
		}

		var procData ProcData
		item.Value(func(valueBuffer []byte) error {
			return json.Unmarshal(valueBuffer, &procData)
		})

		// TODO: remove mutation?
		procData.Status = newStatus

		newValueBytes, err := encodeJSON(procData)
		if err != nil {
			return err
		}

		if err := tx.Set(key, newValueBytes); err != nil {
			return fmt.Errorf("writing procID=%v failed: %w", procID, err)
		}

		return nil
	})
}

func (handle DBHandle) Delete(procIDs []uint64) error {
	return handle.Update(func(tx *badger.Txn) error {
		for _, procID := range procIDs {
			key := encodeUint64(procID)

			if err := tx.Delete(key); err != nil {
				return fmt.Errorf("deleting procID=%d failed: %w", procID, err)
			}
		}

		return nil
	})
}

func (handle DBHandle) Init() error {
	// TODO: any seeding?
	return nil
}

// mex - find (m)inimal number (ex)cluded from set
func mex(set map[uint64]struct{}) uint64 {
	for i := uint64(0); ; i++ {
		if _, ok := set[i]; !ok {
			return i
		}
	}
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
