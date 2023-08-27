package core

import (
	"strconv"

	"github.com/rprtr258/fun"
	"github.com/samber/lo"
)

type Filter struct {
	Names          []string
	Tags           []string
	IDs            []ProcID
	AllIfNoFilters bool
}

func (f Filter) NoFilters() bool {
	return len(f.Names) == 0 &&
		len(f.Tags) == 0 &&
		len(f.IDs) == 0
}

type FilterOption func(*Filter)

func WithGeneric(args []string) FilterOption {
	ids := fun.FilterMap[ProcID](args, func(id string, _ int) (ProcID, bool) {
		procID, err := strconv.ParseUint(id, 10, 64)
		return procID, err == nil
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

func WithIDs(ids ...ProcID) FilterOption {
	return func(cfg *Filter) {
		cfg.IDs = append(cfg.IDs, ids...)
	}
}

func WithAllIfNoFilters(cfg *Filter) {
	cfg.AllIfNoFilters = true
}

func NewFilter(opts ...FilterOption) Filter {
	var filter Filter
	for _, opt := range opts {
		opt(&filter)
	}
	return filter
}

func FilterProcMap(procs map[ProcID]Proc, filter Filter) []ProcID {
	if filter.NoFilters() {
		return fun.Keys(procs)
	}

	return fun.MapFilterToSlice(
		procs,
		func(procID ProcID, proc Proc) (ProcID, bool) {
			return procID, fun.Contains(filter.Names, proc.Name) ||
				lo.Some(filter.Tags, proc.Tags) ||
				fun.Contains(filter.IDs, proc.ID)
		})
}
