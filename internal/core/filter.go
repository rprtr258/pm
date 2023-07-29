package core

import (
	"strconv"

	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core/fun"
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
	ids := lo.FilterMap(args, func(id string, _ int) (ProcID, bool) {
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

func filterProc(proc Proc, filter Filter) bool {
	if filter.NoFilters() {
		return filter.AllIfNoFilters
	}

	return lo.Contains(filter.Names, proc.Name) ||
		lo.Some(filter.Tags, proc.Tags) ||
		lo.Contains(filter.IDs, proc.ID)
}

func FilterProcMap[T ~uint64](procs map[ProcID]Proc, filter Filter) []T {
	return fun.FilterMapToSlice(
		procs,
		func(procID ProcID, proc Proc) (T, bool) {
			return T(procID), filterProc(proc, filter)
		},
	)
}
