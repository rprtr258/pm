package pm

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"

	pb "github.com/rprtr258/pm/api"
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
		if !errors.Is(errConfig, core.ErrConfigNotExists) {
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
) (map[core.ProcID]core.Proc, error) {
	list, errList := app.client.List(ctx)
	if errList != nil {
		return nil, xerr.NewWM(errList, "ListByRunConfigs: list procs")
	}

	procNames := lo.FilterMap(runConfigs, func(cfg core.RunConfig, _ int) (string, bool) {
		return cfg.Name.Unpack()
	})

	configList := lo.PickBy(list, func(_ core.ProcID, procData core.Proc) bool {
		return lo.Contains(procNames, procData.Name)
	})

	return configList, nil
}

func (app App) Signal(
	ctx context.Context,
	signal syscall.Signal,
	procs map[core.ProcID]core.Proc,
	args, names, tags []string, ids []uint64, // TODO: extract to filter struct
) ([]core.ProcID, error) {
	procIDs := core.FilterProcMap[core.ProcID](
		procs,
		core.NewFilter(
			core.WithGeneric(args),
			core.WithIDs(lo.Map(ids, func(id uint64, _ int) core.ProcID {
				return core.ProcID(id)
			})...),
			core.WithNames(names),
			core.WithTags(tags),
			core.WithAllIfNoFilters,
		),
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
	procIDs ...core.ProcID,
) ([]core.ProcID, error) {
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

func (app App) List(ctx context.Context) (map[core.ProcID]core.Proc, error) {
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
	var merr error
	requests := make([]*pb.ProcessOptions, 0, len(configs))
	for _, config := range configs {
		command, errLook := exec.LookPath(config.Command)
		if errLook != nil {
			xerr.AppendInto(&merr, xerr.NewWM(
				errLook,
				"look for executable path",
				xerr.Fields{"executable": config.Command},
			))
			continue
		}
		if command == config.Command { // command contains slash and might be relative
			var errAbs error
			command, errAbs = filepath.Abs(command)
			if errAbs != nil {
				xerr.AppendInto(&merr, xerr.NewWM(
					errAbs,
					"abs",
					xerr.Fields{"command": command},
				))
			}
		}

		requests = append(requests, &pb.ProcessOptions{
			Command: command,
			Args:    config.Args,
			Name:    config.Name.Ptr(),
			Cwd:     config.Cwd,
			Tags:    config.Tags,
			Env:     config.Env,
			Watch: fun.OptMap(config.Watch, func(r *regexp.Regexp) string {
				return r.String()
			}).Ptr(),
			StdoutFile: config.StdoutFile.Ptr(),
			StderrFile: config.StdoutFile.Ptr(),
		})
	}

	procIDs, errCreate := app.client.Create(ctx, requests)
	if errCreate != nil {
		xerr.AppendInto(&merr, xerr.NewWM(
			errCreate,
			"server.create",
			xerr.Fields{"processOptions": requests},
		))
	}

	createdProcIDs := lo.Map(procIDs, func(procID uint64, _ int) core.ProcID {
		return core.ProcID(procID)
	})

	if errStart := app.client.Start(ctx, procIDs); errStart != nil {
		return createdProcIDs, xerr.NewWM(errStart, "start processes", xerr.Errors{merr})
	}

	return createdProcIDs, merr
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

// Logs - watch for processes logs
func (app App) Logs(ctx context.Context, ids ...core.ProcID) (<-chan core.ProcLogs, error) {
	iter, errLogs := app.client.Logs(ctx, lo.Map(ids, func(procID core.ProcID, _ int) uint64 {
		return uint64(procID)
	})...)
	if errLogs != nil {
		return nil, xerr.NewWM(errLogs, "start processes")
	}

	res := make(chan core.ProcLogs)
	go func() {
		defer close(res)

		for {
			select {
			case <-ctx.Done():
				return
			case errIter := <-iter.Err:
				slog.Error("failed to receive log line", slog.Any("err", errIter))
				return
			case procLogs := <-iter.Logs:
				res <- core.ProcLogs{
					ID: core.ProcID(procLogs.GetId()),
					Lines: fun.Map(procLogs.GetLines(), func(line *pb.LogLine) core.LogLine {
						return core.LogLine{
							At:   line.GetTime().AsTime(),
							Line: line.GetLine(),
							Type: lo.
								Switch[pb.LogLine_Type, core.LogType](line.GetType()).
								Case(pb.LogLine_TYPE_STDOUT, core.LogTypeStdout).
								Case(pb.LogLine_TYPE_STDERR, core.LogTypeStderr).
								Default(core.LogTypeUnspecified),
						}
					}),
				}
			}
		}
	}()

	return res, nil
}
