package core

import (
	"github.com/rprtr258/fun"
	"github.com/samber/lo"
)

type Filter struct {
	Names          []string
	Tags           []string
	IDs            []PMID
	AllIfNoFilters bool
}

func (f Filter) NoFilters() bool {
	return len(f.Names) == 0 &&
		len(f.Tags) == 0 &&
		len(f.IDs) == 0
}

type FilterOption func(*Filter)

func WithGeneric(args []string) FilterOption {
	ids := fun.FilterMap[PMID](args, func(id string, _ int) (PMID, bool) {
		isHex := true
		for _, c := range id {
			isHex = isHex && ('0' <= c && c <= '9' || 'a' <= c && c <= 'f')
		}
		return PMID(id), isHex && len(id) == 16*2
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

func WithIDs[ID interface{ ~string }](ids ...ID) FilterOption {
	return func(cfg *Filter) {
		for _, id := range ids {
			cfg.IDs = append(cfg.IDs, PMID(id))
		}
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

func FilterProcMap(procs map[PMID]Proc, filter Filter) []PMID {
	if filter.NoFilters() {
		return fun.Keys(procs)
	}

	return fun.MapFilterToSlice(
		procs,
		func(id PMID, proc Proc) (PMID, bool) {
			return id, fun.Contains(filter.Names, proc.Name) ||
				lo.Some(filter.Tags, proc.Tags) ||
				fun.Contains(filter.IDs, proc.ID)
		})
}
