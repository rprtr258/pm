package pm

import (
	"context"
	"errors"
	"os/exec"
	"syscall"

	"github.com/rprtr258/xerr"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/pkg/client"
)

type App struct {
	client client.Client
	config core.Config
}

func New(client client.Client) (App, error) {
	config, errConfig := core.ReadConfig()
	if errConfig != nil {
		if !errors.Is(errConfig, core.ErrConfigFileNotExists) {
			return App{}, xerr.NewWM(errConfig, "read app config")
		}

		config = core.DefaultConfig
	}

	return App{
		client: client,
		config: config,
	}, nil
}

func (app App) CheckDaemon(ctx context.Context) error {
	if errHealth := app.client.HealthCheck(ctx); errHealth != nil {
		return xerr.NewWM(errHealth, "check daemon health")
	}

	return nil
}

func (app App) ListByRunConfigs(
	ctx context.Context, runConfigs []core.RunConfig,
) (map[core.ProcID]core.ProcData, error) {
	list, errList := app.client.List(ctx)
	if errList != nil {
		return nil, xerr.NewWM(errList, "ListByRunConfigs: list procs")
	}

	procNames := lo.FilterMap(runConfigs, func(cfg core.RunConfig, _ int) (string, bool) {
		return cfg.Name.Unpack()
	})

	configList := lo.PickBy(list, func(_ core.ProcID, procData core.ProcData) bool {
		return lo.Contains(procNames, procData.Name)
	})

	return configList, nil
}

func (app App) Signal(
	ctx context.Context,
	signal syscall.Signal,
	procs map[core.ProcID]core.ProcData,
	args, names, tags []string, ids []uint64, // TODO: extract to filter struct
) ([]core.ProcID, error) {
	procIDs := core.FilterProcs[core.ProcID](
		procs,
		core.WithGeneric(args),
		core.WithIDs(ids),
		core.WithNames(names),
		core.WithTags(tags),
		core.WithAllIfNoFilters,
	)

	if len(procIDs) == 0 {
		return []core.ProcID{}, nil
	}

	if err := app.client.Signal(ctx, signal, lo.Map(procIDs, func(procID core.ProcID, _ int) uint64 {
		return uint64(procID)
	})); err != nil {
		return nil, xerr.NewWM(err, "client.stop")
	}

	return procIDs, nil
}

func (app App) Stop(
	ctx context.Context,
	procs map[core.ProcID]core.ProcData,
	args, names, tags []string, ids []uint64, // TODO: extract to filter struct
) ([]core.ProcID, error) {
	procIDs := core.FilterProcs[core.ProcID](
		procs,
		core.WithGeneric(args),
		core.WithIDs(ids),
		core.WithNames(names),
		core.WithTags(tags),
		core.WithAllIfNoFilters,
	)

	if len(procIDs) == 0 {
		return []core.ProcID{}, nil
	}

	stoppedRawIDs, err := app.client.Stop(ctx, lo.Map(procIDs, func(procID core.ProcID, _ int) uint64 {
		return uint64(procID)
	}))
	stoppedProcIDs := lo.Map(stoppedRawIDs, func(procID uint64, _ int) core.ProcID {
		return core.ProcID(procID)
	})
	if err != nil {
		return stoppedProcIDs, xerr.NewWM(err, "client.stop")
	}

	return stoppedProcIDs, nil
}

func (app App) Delete(ctx context.Context, procIDs ...core.ProcID) ([]core.ProcID, error) {
	if errDelete := app.client.Delete(ctx, lo.Map(procIDs, func(procID core.ProcID, _ int) uint64 {
		return uint64(procID)
	})); errDelete != nil {
		return nil, xerr.NewWM(errDelete, "client.delete", xerr.Fields{"procIDs": procIDs})
	}

	return procIDs, nil
}

func (app App) List(ctx context.Context) (map[core.ProcID]core.ProcData, error) {
	list, errList := app.client.List(ctx)
	if errList != nil {
		return nil, xerr.NewWM(errList, "List: list procs")
	}

	return list, nil
}

// Run - create and start processes, returns ids of created processes.
// ids must be handled before handling error, because it tries to run all
// processes and error contains info about all failed processes, not only first.
func (app App) Run(ctx context.Context, configs ...core.RunConfig) ([]core.ProcID, error) {
	var err error
	requests := make([]*api.ProcessOptions, 0, len(configs))
	for _, config := range configs {
		command, errLook := exec.LookPath(config.Command)
		if errLook != nil {
			xerr.AppendInto(&err, xerr.NewWM(
				errLook,
				"look for executable path",
				xerr.Fields{"executable": config.Command},
			))
			continue
		}

		requests = append(requests, &api.ProcessOptions{
			Command: command,
			Args:    config.Args,
			Name:    config.Name.Ptr(),
			Cwd:     config.Cwd,
			Tags:    config.Tags,
		})
	}

	procIDs, errCreate := app.client.Create(ctx, requests)
	if errCreate != nil {
		xerr.AppendInto(&err, xerr.NewWM(
			errCreate,
			"server.create",
			xerr.Fields{"processOptions": requests},
		))
	}

	createdProcIDs := lo.Map(procIDs, func(procID uint64, _ int) core.ProcID {
		return core.ProcID(procID)
	})

	if errStart := app.client.Start(ctx, procIDs); errStart != nil {
		return createdProcIDs, xerr.NewWM(errStart, "start processes")
	}

	return createdProcIDs, nil
}

// Start already created processes
func (app App) Start(ctx context.Context, ids ...core.ProcID) error {
	if errStart := app.client.Start(ctx, lo.Map(ids, func(procID core.ProcID, _ int) uint64 {
		return uint64(procID)
	})); errStart != nil {
		return xerr.NewWM(errStart, "start processes")
	}

	return nil
}
