package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/samber/lo"
	"go.etcd.io/bbolt"
)

const (
	_mainBucket   = "main"
	_byNameBucket = "by_name"
	_byTagBucket  = "by_tag"
)

type ProcMetadata struct {
	Name   string   `json:"name"`
	Cmd    string   `json:"cmd"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
}

type ProcData struct {
	ID       uint64
	Metadata ProcMetadata
}

type DB struct {
	db bbolt.DB
}

func encodeUintKey(procID uint64) []byte {
	return []byte(strconv.FormatUint(procID, 10))
}

func decodeUintKey(key []byte) (uint64, error) {
	return strconv.ParseUint(string(key), 10, 64)
}

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

// TODO: template key type
func get[V any](bucket *bbolt.Bucket, key []byte) (V, error) {
	// keyBytes, err := encodeUintKey(key)
	bytes := bucket.Get(key)
	if bytes == nil {
		return lo.Empty[V](), fmt.Errorf("value not found by key %v", key)
	}

	return decodeJSON[V](bytes)
}

// TODO: serialize/deserialize protobuffers
// TODO: template key type
func put[V any](bucket *bbolt.Bucket, key []byte, value V) error {
	bytes, err := encodeJSON(value)
	if err != nil {
		return err
	}

	if err := bucket.Put(key, bytes); err != nil {
		return err
	}

	return nil
}

// TODO: store pid in db
func (db *DB) AddTask(metadata ProcMetadata) (uint64, error) {
	var procID uint64
	if err := db.db.Update(func(tx *bbolt.Tx) error {
		{
			mainBucket := tx.Bucket([]byte(_mainBucket))
			if mainBucket == nil {
				return errors.New("main bucket was not found")
			}

			id, err := mainBucket.NextSequence()
			if err != nil {
				return err
			}

			procID = id

			if err := put(mainBucket, encodeUintKey(id), metadata); err != nil {
				return err
			}
		}
		{
			byNameBucket := tx.Bucket([]byte(_byNameBucket))
			if byNameBucket == nil {
				return errors.New("byName bucket was not found")
			}

			idsByName, err := get[[]uint64](byNameBucket, []byte(metadata.Name))
			if err != nil {
				return err
			}

			if err := put(byNameBucket, []byte(metadata.Name), append(idsByName, procID)); err != nil {
				return err
			}
		}
		{
			byTagBucket := tx.Bucket([]byte(_byTagBucket))
			if byTagBucket == nil {
				return errors.New("byTag bucket was not found")
			}

			for _, tag := range metadata.Tags {
				idsByTag, err := get[[]uint64](byTagBucket, []byte(tag))
				if err != nil {
					return err
				}

				if err := put(byTagBucket, []byte(tag), append(idsByTag, procID)); err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return procID, nil
}

func (db *DB) List() ([]ProcData, error) {
	var res []ProcData

	if err := db.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(_mainBucket))
		if bucket == nil {
			return errors.New("main bucket does not exist")
		}

		if err := bucket.ForEach(func(key, value []byte) error {
			id, err := decodeUintKey(key)
			if err != nil {
				return fmt.Errorf("incorrect key: %w", err)
			}

			metadata, err := decodeJSON[ProcMetadata](value)
			if err != nil {
				return err
			}

			res = append(res, ProcData{
				ID: id,
				Metadata: ProcMetadata{
					Name:   metadata.Name,
					Cmd:    metadata.Cmd,
					Status: metadata.Status,
					Tags:   metadata.Tags,
				},
			})

			return nil
		}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (db *DB) SetStatus(procID uint64, newStatus string /*TODO: enum statuses*/) error {
	return db.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(_mainBucket))
		if bucket == nil {
			return errors.New("main bucket does not exist")
		}

		key := encodeUintKey(procID)
		metadata, err := get[ProcMetadata](bucket, key)
		if err != nil {
			return err
		}

		metadata.Status = newStatus

		return put(bucket, key, metadata)
	})
}

func (db *DB) Delete(procID uint64) error {
	return db.db.Update(func(tx *bbolt.Tx) error {
		mainBucket := tx.Bucket([]byte(_mainBucket))
		if mainBucket == nil {
			return errors.New("main bucket was not found")
		}

		key := encodeUintKey(procID)

		proc, err := get[ProcMetadata](mainBucket, key)
		if err != nil {
			return err
		}

		if err := mainBucket.Delete(key); err != nil {
			return err
		}

		{
			byNameBucket := tx.Bucket([]byte(_byNameBucket))
			if byNameBucket == nil {
				return errors.New("byName bucket was not found")
			}

			if err := byNameBucket.Delete([]byte(proc.Name)); err != nil {
				return err
			}
		}
		{
			byTagBucket := tx.Bucket([]byte(_byTagBucket))
			if byTagBucket == nil {
				return errors.New("byTag bucket was not found")
			}

			for _, tag := range proc.Tags {
				idsByTag, err := get[[]uint64](byTagBucket, []byte(tag))
				if err != nil {
					return err
				}

				if err := put(byTagBucket, []byte(tag), lo.Reject(idsByTag, func(item uint64, _ int) bool {
					return item == procID
				})); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
