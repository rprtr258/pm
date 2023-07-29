package core

import (
	"strconv"

	fun2 "github.com/rprtr258/fun"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core/fun"
)

type filter struct {
	Names          []string
	Tags           []string
	IDs            []ProcID
	allIfNoFilters bool
}

type FilterOption func(*filter)

func WithGeneric(args []string) FilterOption {
	ids := lo.FilterMap(args, func(id string, _ int) (ProcID, bool) {
		procID, err := strconv.ParseUint(id, 10, 64)
		return ProcID(procID), err == nil
	})

	return func(cfg *filter) {
		cfg.IDs = append(cfg.IDs, ids...)
		cfg.Names = append(cfg.Names, args...)
		cfg.Tags = append(cfg.Tags, args...)
	}
}

func WithNames(names []string) FilterOption {
	return func(cfg *filter) {
		cfg.Names = append(cfg.Names, names...)
	}
}

func WithTags(tags []string) FilterOption {
	return func(cfg *filter) {
		cfg.Tags = append(cfg.Tags, tags...)
	}
}

func WithIDs(ids []uint64) FilterOption {
	return func(cfg *filter) {
		procIDs := fun2.Map(ids, func(id uint64) ProcID {
			return ProcID(id)
		})
		cfg.IDs = append(cfg.IDs, procIDs...)
	}
}

func WithAllIfNoFilters(cfg *filter) {
	cfg.allIfNoFilters = true
}

func FilterProcs[T ~uint64](procs map[ProcID]Proc, opts ...FilterOption) []T {
	var cfg filter
	for _, opt := range opts {
		opt(&cfg)
	}

	noFilters := len(cfg.Names) == 0 &&
		len(cfg.Tags) == 0 &&
		len(cfg.IDs) == 0
	switch {
	case !noFilters:
		return fun.FilterMapToSlice(procs, func(procID ProcID, proc Proc) (T, bool) {
			return T(procID), lo.Contains(cfg.Names, proc.Name) ||
				lo.Some(cfg.Tags, proc.Tags) ||
				lo.Contains(cfg.IDs, proc.ID)
		})
	case cfg.allIfNoFilters:
		return lo.Map(lo.Keys(procs), func(id ProcID, _ int) T {
			return T(id)
		})
	default:
		return nil
	}
}
