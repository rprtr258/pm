package core

import (
	"strconv"

	fun2 "github.com/rprtr258/fun"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core/fun"
)

type Filter struct {
	Names          []string
	Tags           []string
	IDs            []ProcID
	allIfNoFilters bool
}

type FilterOption func(*Filter)

func WithGeneric(args []string) FilterOption {
	ids := lo.FilterMap(args, func(id string, _ int) (ProcID, bool) {
		procID, err := strconv.ParseUint(id, 10, 64)
		return ProcID(procID), err == nil
	})

	return func(cfg *Filter) {
		cfg.IDs = append(cfg.IDs, ids...)
		cfg.Names = append(cfg.Names, args...)
		cfg.Tags = append(cfg.Tags, args...)
	}
}

func WithNames(names []string) FilterOption {
	return func(cfg *Filter) {
		cfg.Names = append(cfg.Names, names...)
	}
}

func WithTags(tags []string) FilterOption {
	return func(cfg *Filter) {
		cfg.Tags = append(cfg.Tags, tags...)
	}
}

func WithIDs(ids []uint64) FilterOption {
	return func(cfg *Filter) {
		procIDs := fun2.Map(ids, func(id uint64) ProcID {
			return ProcID(id)
		})
		cfg.IDs = append(cfg.IDs, procIDs...)
	}
}

func WithAllIfNoFilters(cfg *Filter) {
	cfg.allIfNoFilters = true
}

func NewFilter(opts ...FilterOption) Filter {
	var filter Filter
	for _, opt := range opts {
		opt(&filter)
	}
	return filter
}

func FilterProcMap[T ~uint64](procs map[ProcID]Proc, filter Filter) []T {
	noFilters := len(filter.Names) == 0 &&
		len(filter.Tags) == 0 &&
		len(filter.IDs) == 0
	switch {
	case !noFilters:
		return fun.FilterMapToSlice(procs, func(procID ProcID, proc Proc) (T, bool) {
			return T(procID), lo.Contains(filter.Names, proc.Name) ||
				lo.Some(filter.Tags, proc.Tags) ||
				lo.Contains(filter.IDs, proc.ID)
		})
	case filter.allIfNoFilters:
		return lo.Map(lo.Keys(procs), func(id ProcID, _ int) T {
			return T(id)
		})
	default:
		return nil
	}
}
