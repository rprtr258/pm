package core

import (
	"log"
	"strconv"

	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core/fun"
)

type filter struct {
	Names    []string
	Tags     []string
	Statuses []StatusType
	IDs      []ProcID
}

type filterConfig struct {
	filter         filter
	allIfNoFilters bool
}

type FilterProcsOption func(*filterConfig)

func WithGeneric(args []string) FilterProcsOption {
	ids := lo.FilterMap(args, func(id string, _ int) (ProcID, bool) {
		procID, err := strconv.ParseUint(id, 10, 64) //nolint:gomnd // parse id as decimal uint64
		if err != nil {
			return 0, false
		}

		return ProcID(procID), true
	})

	return func(cfg *filterConfig) {
		cfg.filter.IDs = append(cfg.filter.IDs, ids...)
		cfg.filter.Names = append(cfg.filter.Names, args...)
		cfg.filter.Tags = append(cfg.filter.Tags, args...)
	}
}

func WithNames(args []string) FilterProcsOption {
	return func(cfg *filterConfig) {
		cfg.filter.Names = append(cfg.filter.Names, args...)
	}
}

func WithTags(args []string) FilterProcsOption {
	return func(cfg *filterConfig) {
		cfg.filter.Tags = append(cfg.filter.Tags, args...)
	}
}

func WithStatuses(args []string) FilterProcsOption {
	statuses := lo.Map(args, func(status string, _ int) StatusType {
		switch status {
		case "invalid":
			return StatusInvalid
		case "starting":
			return StatusStarting
		case "running":
			return StatusRunning
		case "stopped":
			return StatusStopped
		default:
			log.Printf("unknown status %q\n", status)
			return lo.Empty[StatusType]()
		}
	})

	return func(cfg *filterConfig) {
		cfg.filter.Statuses = append(cfg.filter.Statuses, statuses...)
	}
}

func WithIDs[T uint64](args []T) FilterProcsOption {
	return func(cfg *filterConfig) {
		cfg.filter.IDs = append(cfg.filter.IDs, lo.Map(args, func(id T, _ int) ProcID {
			return ProcID(id)
		})...)
	}
}

func WithAllIfNoFilters(cfg *filterConfig) {
	cfg.allIfNoFilters = true
}

func FilterProcs[T ~uint64](procs map[ProcID]ProcData, opts ...FilterProcsOption) []T {
	var cfg filterConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	// if no filters, return all
	if len(cfg.filter.Names) == 0 &&
		len(cfg.filter.Tags) == 0 &&
		len(cfg.filter.Statuses) == 0 &&
		len(cfg.filter.IDs) == 0 {
		if !cfg.allIfNoFilters {
			return nil
		}

		return lo.Map(lo.Keys(procs), func(id ProcID, _ int) T {
			return T(id)
		})
	}

	return fun.FilterMapToSlice(procs, func(procID ProcID, proc ProcData) (T, bool) {
		return T(procID), lo.Contains(cfg.filter.Names, proc.Name) ||
			lo.Some(cfg.filter.Tags, proc.Tags) ||
			lo.Contains(cfg.filter.Statuses, proc.Status.Status) ||
			lo.Contains(cfg.filter.IDs, proc.ProcID)
	})
}
