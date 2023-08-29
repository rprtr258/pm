package pm

import (
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/pkg/client"
)

type App struct {
	client client.Client
	config core.Config
}

func New(pmClient client.Client) (App, error) {
	config, errConfig := core.ReadConfig()
	if errConfig != nil {
		if errConfig == core.ErrConfigNotExists {
			return App{
				client: pmClient,
				config: core.DefaultConfig,
			}, nil
		}

		return App{}, xerr.NewWM(errConfig, "read app config")
	}

	return App{
		client: pmClient,
		config: config,
	}, nil
}

func (app App) CheckDaemon(ctx context.Context) error {
	if errHealth := app.client.HealthCheck(ctx); errHealth != nil {
		return xerr.NewWM(errHealth, "check daemon health")
	}

	return nil
}

func (app App) ListByRunConfigs(ctx context.Context, runConfigs []core.RunConfig) (core.Procs, error) {
	list, errList := app.client.List(ctx)
	if errList != nil {
		return nil, xerr.NewWM(errList, "ListByRunConfigs: list procs")
	}

	procNames := fun.FilterMap[string](runConfigs, func(cfg core.RunConfig) fun.Option[string] {
		return cfg.Name
	})

	configList := lo.PickBy(list, func(_ core.ProcID, procData core.Proc) bool {
		return fun.Contains(procNames, procData.Name)
	})

	return configList, nil
}

func (app App) Signal(
	ctx context.Context,
	signal syscall.Signal,
	procIDs ...core.ProcID,
) ([]core.ProcID, error) {
	if len(procIDs) == 0 {
		return []core.ProcID{}, nil
	}

	for _, id := range procIDs {
		if err := app.client.Signal(ctx, signal, id); err != nil {
			return nil, xerr.NewWM(err, "client.stop", xerr.Fields{"proc_id": id})
		}
	}

	return procIDs, nil
}

func (app App) Stop(ctx context.Context, procIDs ...core.ProcID) error {
	for _, id := range procIDs {
		if err := app.client.Stop(ctx, id); err != nil {
			return xerr.NewWM(err, "client.stop", xerr.Fields{"proc_id": id})
		}
	}

	return nil
}

func (app App) Delete(ctx context.Context, procIDs ...core.ProcID) error {
	for _, id := range procIDs {
		if errDelete := app.client.Delete(ctx, id); errDelete != nil {
			return xerr.NewWM(errDelete, "client.delete", xerr.Fields{"proc_id": id})
		}
	}

	return nil
}

func (app App) List(ctx context.Context) (core.Procs, error) {
	list, errList := app.client.List(ctx)
	if errList != nil {
		return nil, xerr.NewWM(errList, "List: list procs")
	}

	return list, nil
}

// Run - create and start processes, returns ids of created processes.
// ids must be handled before handling error, because it tries to run all
// processes and error contains info about all failed processes, not only first.
func (app App) Run(ctx context.Context, config core.RunConfig) (core.ProcID, error) {
	var merr error
	command, errLook := exec.LookPath(config.Command)
	if errLook != nil {
		return 0, xerr.NewWM(
			errLook,
			"look for executable path",
			xerr.Fields{"executable": config.Command},
		)
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

	request := &pb.CreateRequest{
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
	}
	createdProcIDs, errCreate := app.client.Create(ctx, request)
	if errCreate != nil {
		return 0, xerr.NewWM(
			errCreate,
			"server.create",
			xerr.Fields{"process_options": request},
		)
	}

	if errStart := app.client.Start(ctx, createdProcIDs); errStart != nil {
		return createdProcIDs, xerr.NewWM(errStart, "start processes", xerr.Errors{merr})
	}

	return createdProcIDs, merr
}

// Start already created processes
func (app App) Start(ctx context.Context, ids ...core.ProcID) error {
	for _, id := range ids {
		if errStart := app.client.Start(ctx, id); errStart != nil {
			return xerr.NewWM(errStart, "start processes")
		}
	}

	return nil
}

// Logs - watch for processes logs
func (app App) Logs(ctx context.Context, id core.ProcID) (<-chan core.ProcLogs, error) {
	iterr, errLogs := app.client.Logs(ctx, id)
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
			case errIter := <-iterr.Err:
				log.Error().Err(errIter).Msg("failed to receive log line")
				return
			case procLogs := <-iterr.Logs:
				res <- core.ProcLogs{
					ID: procLogs.GetId(),
					Lines: fun.Map[core.LogLine](
						procLogs.GetLines(),
						func(line *pb.LogLine, _ int) core.LogLine {
							return core.LogLine{
								At:   line.GetTime().AsTime(),
								Line: line.GetLine(),
								Type: map[pb.LogLine_Type]core.LogType{
									pb.LogLine_TYPE_STDOUT:      core.LogTypeStdout,
									pb.LogLine_TYPE_STDERR:      core.LogTypeStderr,
									pb.LogLine_TYPE_UNSPECIFIED: core.LogTypeUnspecified,
								}[line.GetType()],
							}
						}),
				}
			}
		}
	}()

	return res, nil
}
