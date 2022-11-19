package internal

import (
	"log"
	"strconv"

	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/db"
)

type Filter struct {
	Generic  []string
	Names    []string
	Tags     []string
	Statuses []db.ProcStatus
	IDs      []db.ProcID
}

type filterConfig struct {
	filter         Filter
	allIfNoFilters bool
}

type filterOption func(*filterConfig)

func WithGeneric(args []string) filterOption {
	return func(cfg *filterConfig) {
		cfg.filter.Generic = append(cfg.filter.Generic, args...)
	}
}

func WithNames(args []string) filterOption {
	return func(cfg *filterConfig) {
		cfg.filter.Names = append(cfg.filter.Names, args...)
	}
}

func WithTags(args []string) filterOption {
	return func(cfg *filterConfig) {
		cfg.filter.Tags = append(cfg.filter.Tags, args...)
	}
}

func WithStatuses(args []string) filterOption {
	statuses := lo.Map(args, func(status string, _ int) db.ProcStatus {
		switch status {
		case "invalid":
			return db.StatusInvalid
		case "starting":
			return db.StatusStarting
		case "running":
			return db.StatusRunning
		case "stopped":
			return db.StatusStopped
		case "errored":
			return db.StatusErrored
		default:
			log.Printf("unknown status %q\n", status)
			return lo.Empty[db.ProcStatus]()
		}
	})

	return func(cfg *filterConfig) {
		cfg.filter.Statuses = append(cfg.filter.Statuses, statuses...)
	}
}

func WithIDs(args []uint64) filterOption {
	return func(cfg *filterConfig) {
		cfg.filter.IDs = append(cfg.filter.IDs, lo.Map(args, func(id uint64, _ int) db.ProcID {
			return db.ProcID(id)
		})...)
	}
}

func WithAllIfNoFilters(cfg *filterConfig) {
	cfg.allIfNoFilters = true
}

// TODO: options api
func FilterProcs(procs db.DB, opts ...filterOption) []db.ProcID {
	cfg := filterConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	// if no filters, return all
	if len(cfg.filter.Generic) == 0 &&
		len(cfg.filter.Names) == 0 &&
		len(cfg.filter.Tags) == 0 &&
		len(cfg.filter.Statuses) == 0 &&
		len(cfg.filter.IDs) == 0 {
		if cfg.allIfNoFilters {
			return lo.Keys(procs)
		} else {
			return nil
		}
	}

	genericIDs := lo.FilterMap(cfg.filter.Generic, func(filter string, _ int) (db.ProcID, bool) {
		id, err := strconv.ParseUint(filter, 10, 64)
		if err != nil {
			return 0, false
		}

		return db.ProcID(id), true
	})

	return FilterMapToSlice(procs, func(procID db.ProcID, proc db.ProcData) (db.ProcID, bool) {
		return procID, lo.Contains(cfg.filter.Names, proc.Name) ||
			lo.Some(cfg.filter.Tags, proc.Tags) ||
			lo.Contains(cfg.filter.Statuses, proc.Status.Status) ||
			lo.Contains(cfg.filter.IDs, proc.ID) ||
			lo.Contains(cfg.filter.Generic, proc.Name) ||
			lo.Some(cfg.filter.Generic, proc.Tags) ||
			lo.Contains(genericIDs, proc.ID)
	})
}
