package db

import (
	"fmt"

	"github.com/rprtr258/simpdb"
	"github.com/rprtr258/simpdb/storages"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core"
)

type Handle struct {
	db    *simpdb.DB
	procs *simpdb.Table[core.ProcData]
}

func New(dir string) (Handle, error) {
	db := simpdb.New(dir)

	procs, err := simpdb.GetTable[core.ProcData](db, "procs", storages.NewJSONStorage[core.ProcData]())
	if err != nil {
		return Handle{}, err
	}

	return Handle{
		db:    db,
		procs: procs,
	}, nil
}

func (handle Handle) AddProc(metadata core.ProcData) (core.ProcID, error) {
	maxProcID := core.ProcID(0)
	handle.procs.Iter(func(_ string, proc core.ProcData) bool {
		if proc.ProcID > maxProcID {
			maxProcID = proc.ProcID
		}

		return true
	})

	// TODO: remove mutation?
	metadata.ProcID = maxProcID + 1

	if !handle.procs.Insert(metadata) {
		return 0, xerr.NewM("insert: already present")
	}

	if err := handle.procs.Flush(); err != nil {
		return 0, xerr.NewWM(err, "insert: db flush")
	}

	return metadata.ProcID, nil
}

func (handle Handle) UpdateProc(metadata core.ProcData) error {
	handle.procs.Upsert(metadata)

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "update: db flush")
	}

	return nil
}

func (handle Handle) GetProcs(ids []core.ProcID) ([]core.ProcData, error) {
	lookupTable := lo.SliceToMap(ids, func(id core.ProcID) (string, struct{}) {
		return id.String(), struct{}{}
	})

	return handle.procs.
		Where(func(id string, _ core.ProcData) bool {
			_, ok := lookupTable[id]
			return ok
		}).
		List().
		All(), nil
}

func (handle Handle) List() map[core.ProcID]core.ProcData {
	res := map[core.ProcID]core.ProcData{}
	handle.procs.Iter(func(id string, pd core.ProcData) bool {
		res[pd.ProcID] = pd
		return true
	})
	return res
}

type ProcNotFoundError core.ProcID

func (err ProcNotFoundError) Error() string {
	return fmt.Sprintf("proc #%d not found", err)
}

func (handle Handle) SetStatus(procID core.ProcID, newStatus core.Status) error {
	procDataMaybe := handle.procs.Get(procID.String())
	if !procDataMaybe.Valid {
		return ProcNotFoundError(procID)
	}

	procDataMaybe.Value.Status = newStatus
	handle.procs.Upsert(procDataMaybe.Value)

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "set status: db flush")
	}

	return nil
}

func (handle Handle) Delete(procIDs []uint64) error {
	lookupTable := lo.SliceToMap(procIDs, func(id uint64) (core.ProcID, struct{}) {
		return core.ProcID(id), struct{}{}
	})

	handle.procs.Where(func(_ string, pd core.ProcData) bool {
		_, ok := lookupTable[pd.ProcID]
		return ok
	}).Delete()

	if err := handle.procs.Flush(); err != nil {
		return xerr.NewWM(err, "delete: db flush")
	}

	return nil
}
