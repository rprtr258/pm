package pm

import (
	"context"
	"fmt"

	"github.com/rprtr258/xerr"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
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

func (app App) ListByRunConfigs(ctx context.Context, runConfigs []core.RunConfig) (map[db.ProcID]db.ProcData, error) {
	list, errList := app.client.List(ctx)
	if errList != nil {
		return nil, xerr.NewWM(errList, "ListByRunConfigs: list procs")
	}

	procNames := lo.FilterMap(runConfigs, func(cfg core.RunConfig, _ int) (string, bool) {
		return cfg.Name.Unpack()
	})

	configList := lo.PickBy(list, func(_ db.ProcID, procData db.ProcData) bool {
		return lo.Contains(procNames, procData.Name)
	})

	return configList, nil
}

func (app App) Delete(
	ctx context.Context,
	procs map[db.ProcID]db.ProcData,
	args, names, tags []string, ids []uint64, // TODO: extract to filter struct
) error {
	procIDs := core.FilterProcs[uint64](
		procs,
		core.WithGeneric(args),
		core.WithIDs(ids),
		core.WithNames(names),
		core.WithTags(tags),
		core.WithAllIfNoFilters,
	)

	if len(procIDs) == 0 {
		fmt.Println("Nothing to stop, leaving")
		return nil
	}

	fmt.Printf("Stopping: %v\n", procIDs)

	if err := app.client.Stop(ctx, procIDs); err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	fmt.Printf("Removing: %v\n", procIDs)

	if errDelete := app.client.Delete(ctx, procIDs); errDelete != nil {
		return xerr.NewWM(errDelete, "client.delete", xerr.Fields{"procIDs": procIDs})
	}

	return nil
}
