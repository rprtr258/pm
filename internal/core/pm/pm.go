package pm

import (
	"context"
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
}

func New(client client.Client) App {
	return App{
		client: client,
	}
}

func (app App) CheckDaemon(ctx context.Context) error {
	if errHealth := app.client.HealthCheck(ctx); errHealth != nil {
		return xerr.NewWM(errHealth, "check daemon health")
	}

	return nil
}

func (app App) ListByRunConfigs(ctx context.Context, runConfigs []core.RunConfig) (map[core.ProcID]core.ProcData, error) {
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

func (app App) Stop(
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

// Run - create processes, returns ids of created processes.
// ids must be handled before handling error, because it tries to run all
// processes and error contains info about all failed processes, not only first.
func (app App) Create(ctx context.Context, configs ...core.RunConfig) ([]core.ProcID, error) {
	var err error
	procIDs := make([]core.ProcID, 0, len(configs))
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

		req := &api.ProcessOptions{
			Command: command,
			Args:    config.Args,
			Name:    config.Name.Ptr(),
			Cwd:     config.Cwd,
			Tags:    config.Tags,
		}
		procID, errCreate := app.client.Create(ctx, req)
		if errCreate != nil {
			xerr.AppendInto(&err, xerr.NewWM(
				errCreate,
				"server.create",
				xerr.Fields{"processOptions": req},
			))
			continue
		}

		procIDs = append(procIDs, core.ProcID(procID))
	}

	return procIDs, err
}

func (app App) Start(ctx context.Context, procIDs ...core.ProcID) error {
	procIDs2 := lo.Map(procIDs, func(procID core.ProcID, _ int) uint64 {
		return uint64(procID)
	})

	if errStart := app.client.Start(ctx, procIDs2); errStart != nil {
		return xerr.NewWM(errStart, "start processes")
	}

	return nil
}
