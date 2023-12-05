package core

import (
	"unsafe"

	"github.com/rprtr258/fun"
	"github.com/samber/lo"
)

type filter struct {
	Names          []string
	Tags           []string
	IDs            []PMID
	AllIfNoFilters bool
}

func (f filter) NoFilters() bool {
	return len(f.Names) == 0 &&
		len(f.Tags) == 0 &&
		len(f.IDs) == 0
}

type FilterOption func(*filter)

func reinterpretSlice[R, T any](slice []T) []R {
	return *(*[]R)(unsafe.Pointer(&slice))
}

func WithGeneric[S ~string](args ...S) FilterOption {
	ids := fun.FilterMap[PMID](func(id S, _ int) (PMID, bool) {
		isHex := true
		for _, c := range id {
			isHex = isHex && ('0' <= c && c <= '9' || 'a' <= c && c <= 'f')
		}
		return PMID(id), isHex && len(id) == 16*2
	}, args...)

	return func(cfg *filter) {
		cfg.IDs = append(cfg.IDs, ids...)
		cfg.Names = append(cfg.Names, reinterpretSlice[string](args)...)
		cfg.Tags = append(cfg.Tags, reinterpretSlice[string](args)...)
	}
}

func WithNames[S ~string](names ...S) FilterOption {
	return func(cfg *filter) {
		cfg.Names = append(cfg.Names, reinterpretSlice[string](names)...)
	}
}

func WithTags[S ~string](tags ...S) FilterOption {
	return func(cfg *filter) {
		cfg.Tags = append(cfg.Tags, reinterpretSlice[string](tags)...)
	}
}

func WithIDs[S ~string](ids ...S) FilterOption {
	return func(cfg *filter) {
		for _, id := range ids {
			cfg.IDs = append(cfg.IDs, PMID(id))
		}
	}
}

func WithAllIfNoFilters(cfg *filter) {
	cfg.AllIfNoFilters = true
}

func FilterFunc(opts ...FilterOption) func(Proc) bool {
	var _filter filter
	for _, opt := range opts {
		opt(&_filter)
	}

	if _filter.NoFilters() {
		return func(Proc) bool {
			return _filter.AllIfNoFilters
		}
	}

	return func(proc Proc) bool {
		return fun.Contains(proc.Name, _filter.Names...) ||
			lo.Some(_filter.Tags, proc.Tags) ||
			fun.Contains(proc.ID, _filter.IDs...)
	}
}

func FilterProcMap(procs map[PMID]Proc, opts ...FilterOption) []PMID {
	f := FilterFunc(opts...)

	return fun.MapFilterToSlice(
		procs,
		func(id PMID, proc Proc) (PMID, bool) {
			return id, f(proc)
		})
}
