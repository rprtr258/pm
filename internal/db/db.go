package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/samber/lo"
	"go.etcd.io/bbolt"
)

type ProcStatus int

const (
	StatusInvalid ProcStatus = iota
	StatusStarting
	StatusRunning
	StatusStopped
	StatusErrored
)

var _mainBucket = []byte("main")

type Status struct {
	Status ProcStatus `json:"status"`
	// nulls if not running
	Pid       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
	Cpu       uint64    `json:"cpu"`    // round(cpu usage in % * 100)
	Memory    uint64    `json:"memory"` // in bytes
}

type ProcID uint64

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

func encodeUintKey(procID uint64) []byte {
	return []byte(strconv.FormatUint(procID, 10))
}

// func decodeUintKey(key []byte) (uint64, error) {
// 	return strconv.ParseUint(string(key), 10, 64)
// }

func encodeJSON[T any](proc T) ([]byte, error) {
	return json.Marshal(proc)
}

func decodeJSON[T any](value []byte) (T, error) {
	var res T
	if err := json.Unmarshal(value, &res); err != nil {
		return lo.Empty[T](), fmt.Errorf("failed decoding value of type %T: %w", res, err)
	}

	return res, nil
}

func get[V any](bucket *bbolt.Bucket, key []byte) (V, error) {
	// keyBytes, err := encodeUintKey(key)
	bytes := bucket.Get(key)
	if bytes == nil {
		return lo.Empty[V](), nil // fmt.Errorf("value not found by key %v", key)
	}

	return decodeJSON[V](bytes)
}

// TODO: serialize/deserialize protobuffers
func put[V any](bucket *bbolt.Bucket, key []byte, value V) error {
	bytes, err := encodeJSON(value)
	if err != nil {
		return err
	}

	if err := bucket.Put(key, bytes); err != nil {
		return fmt.Errorf("bucket.Put failed: %w", err)
	}

	return nil
}

func New(dbFile string) DBHandle {
	return DBHandle{
		dbFilename: dbFile,
	}
}

func (handle DBHandle) execute(statement func(*bbolt.DB) error) error {
	db, err := bbolt.Open(handle.dbFilename, 0600, nil)
	if err != nil {
		return fmt.Errorf("bbolt.Open failed: %w", err)
	}
	defer db.Close()

	return statement(db)
}

func (handle DBHandle) Update(statement func(*bbolt.Tx) error) error {
	return handle.execute(func(db *bbolt.DB) error {
		return db.Update(func(tx *bbolt.Tx) error {
			return statement(tx)
		})
	})
}

func (handle DBHandle) View(statement func(*bbolt.Tx) error) error {
	return handle.execute(func(db *bbolt.DB) error {
		return db.View(func(tx *bbolt.Tx) error {
			return statement(tx)
		})
	})
}

func (handle DBHandle) AddProc(metadata ProcData) (ProcID, error) {
	var procID uint64

	err := handle.Update(func(tx *bbolt.Tx) error {
		mainBucket := tx.Bucket(_mainBucket)
		if mainBucket == nil {
			return errors.New("main bucket was not found")
		}

		// TODO: manage ids myself
		id, err := mainBucket.NextSequence()
		if err != nil {
			return fmt.Errorf("bucket.NextSequence failed: %w", err)
		}

		procID = id
		metadata.ID = ProcID(id)

		if err := put(mainBucket, encodeUintKey(id), metadata); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("db.AddProc failed: %w", err)
	}

	return ProcID(procID), nil
}

func (handle DBHandle) GetProcs(ids []ProcID) ([]ProcData, error) {
	var res []ProcData

	err := handle.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(_mainBucket)
		if bucket == nil {
			return errors.New("main bucket does not exist")
		}

		if err := bucket.ForEach(func(_, value []byte) error {
			procData, err := decodeJSON[ProcData](value)
			if err != nil {
				return err
			}

			if !lo.Contains(ids, procData.ID) {
				return nil
			}

			res = append(res, procData)

			return nil
		}); err != nil {
			return fmt.Errorf("bucket.ForEach failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("db.GetProcs failed: %w", err)
	}

	return res, nil
}

func (handle DBHandle) List() ([]ProcData, error) {
	var res []ProcData

	err := handle.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(_mainBucket)
		if bucket == nil {
			return errors.New("main bucket does not exist")
		}

		if err := bucket.ForEach(func(_, value []byte) error {
			procData, err := decodeJSON[ProcData](value)
			if err != nil {
				return err
			}

			res = append(res, procData)

			return nil
		}); err != nil {
			return fmt.Errorf("bucket.ForEach failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("db.List failed: %w", err)
	}

	return res, nil
}

func (handle DBHandle) SetStatus(procID ProcID, newStatus Status) error {
	return handle.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(_mainBucket)
		if bucket == nil {
			return errors.New("main bucket does not exist")
		}

		key := encodeUintKey(uint64(procID))
		metadata, err := get[ProcData](bucket, key)
		if err != nil {
			return fmt.Errorf("set status failed: %w", err)
		}

		metadata.Status = newStatus

		return put(bucket, key, metadata)
	})
}

func (handle DBHandle) Delete(procIDs []uint64) error {
	return handle.Update(func(tx *bbolt.Tx) error {
		mainBucket := tx.Bucket(_mainBucket)
		if mainBucket == nil {
			return errors.New("main bucket was not found")
		}

		for _, procID := range procIDs {
			key := encodeUintKey(procID)

			// proc, err := get[ProcData](mainBucket, key)
			// if err != nil {
			// 	return fmt.Errorf("failed getting proc with id %d: %w", procID, err)
			// }

			if err := mainBucket.Delete(key); err != nil {
				return fmt.Errorf("failed deleting proc with id %d: %w", procID, err)
			}
		}

		return nil
	})
}

// Init - init buckets
func (handle DBHandle) Init() error {
	return handle.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(_mainBucket); err != nil {
			return fmt.Errorf("bucket creating failed: %w", err)
		}

		return nil
	})
}
