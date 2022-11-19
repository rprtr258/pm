package internal

import (
	"strconv"

	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/db"
)

// TODO: options api
func FilterProcs(
	procs db.DB,
	generic, names, tags []string,
	statuses []db.ProcStatus,
	ids []db.ProcID,
) []db.ProcID {
	// if no filters, return all
	if len(generic) == 0 &&
		len(names) == 0 &&
		len(tags) == 0 &&
		len(statuses) == 0 &&
		len(ids) == 0 {
		return lo.Keys(procs)
	}

	genericIDs := lo.FilterMap(generic, func(filter string, _ int) (db.ProcID, bool) {
		id, err := strconv.ParseUint(filter, 10, 64)
		if err != nil {
			return 0, false
		}

		return db.ProcID(id), true
	})

	return FilterMapToSlice(procs, func(procID db.ProcID, proc db.ProcData) (db.ProcID, bool) {
		return procID, lo.Contains(names, proc.Name) ||
			lo.Some(tags, proc.Tags) ||
			lo.Contains(statuses, proc.Status.Status) ||
			lo.Contains(ids, proc.ID) ||
			lo.Contains(generic, proc.Name) ||
			lo.Some(generic, proc.Tags) ||
			lo.Contains(genericIDs, proc.ID)
	})
}
