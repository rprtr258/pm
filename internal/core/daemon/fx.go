package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/runner"
	"github.com/rprtr258/pm/internal/core/daemon/watcher"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type fxLogger struct{ zerolog.Logger }

func (l fxLogger) LogEvent(e fxevent.Event) {
	switch e := e.(type) {
	case *fxevent.OnStartExecuting:
		l.Info().
			Str("function_name", e.FunctionName).
			Str("caller_name", e.CallerName).
			Msg("start executing")
	case *fxevent.OnStartExecuted:
		l.Info().
			Str("function_name", e.FunctionName).
			Str("caller_name", e.CallerName).
			Str("method", e.Method).
			Dur("runtime", e.Runtime).
			Err(e.Err).
			Msg("start executed")
	case *fxevent.OnStopExecuting:
		l.Info().
			Str("function_name", e.FunctionName).
			Str("caller_name", e.CallerName).
			Msg("stop executing")
	case *fxevent.OnStopExecuted:
		l.Info().
			Str("function_name", e.FunctionName).
			Str("caller_name", e.CallerName).
			Dur("runtime", e.Runtime).
			Err(e.Err).
			Msg("stop executed")
	case *fxevent.Supplied:
		l.Info().
			Str("type_name", e.TypeName).
			Strs("stacktrace", e.StackTrace).
			Str("module_name", e.ModuleName).
			Err(e.Err).
			Msg("supplied")
	case *fxevent.Provided:
		l.Info().
			Str("constructor_name", e.ConstructorName).
			Strs("output_type_names", e.OutputTypeNames).
			Str("module_name", e.ModuleName).
			Err(e.Err).
			Bool("private", e.Private).
			Msg("provided")
	case *fxevent.Replaced:
		l.Info().
			Strs("output_type_names", e.OutputTypeNames).
			Str("module_name", e.ModuleName).
			Err(e.Err).
			Msg("replaced")
	case *fxevent.Decorated:
		l.Info().
			Str("decorator_name", e.DecoratorName).
			Str("module_name", e.ModuleName).
			Strs("output_type_names", e.OutputTypeNames).
			Err(e.Err).
			Msg("decorated")
	case *fxevent.Run:
		l.Info().
			Str("name", e.Name).
			Str("module_name", e.ModuleName).
			Str("kind", e.Kind).
			Err(e.Err).
			Msg("run")
	case *fxevent.Invoking:
		l.Info().
			Str("function_name", e.FunctionName).
			Str("module_name", e.ModuleName).
			Msg("invoking")
	case *fxevent.Invoked:
		l.Info().
			Str("function_name", e.FunctionName).
			Str("module_name", e.ModuleName).
			Str("trace", e.Trace).
			Err(e.Err).
			Msg("invoked")
	case *fxevent.Stopping:
		l.Info().Str("signal", e.Signal.String()).Msg("stopping")
	case *fxevent.Stopped:
		l.Info().Err(e.Err).Msg("stopped")
	case *fxevent.RollingBack:
		l.Info().Err(e.StartErr).Msg("rolling back")
	case *fxevent.RolledBack:
		l.Info().Err(e.Err).Msg("rolled back")
	case *fxevent.Started:
		l.Info().Err(e.Err).Msg("started")
	case *fxevent.LoggerInitialized:
		l.Info().
			Str("constructor_name", e.ConstructorName).
			Err(e.Err).
			Msg("logger initialized")
	default:
		l.Warn().
			Str("event_type", fmt.Sprintf("%#v", e)).
			Msg("unknown event type")
	}
}

// TODO: not working, fix
func startChildrenStatuser(lc fx.Lifecycle, ebus *eventbus.EventBus, dbHandle db.Handle) {
	c := make(chan os.Signal, 10) // arbitrary buffer size
	signal.Notify(c, syscall.SIGCHLD)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				for range c {
					// wait for any of childs' death
					for {
						var status syscall.WaitStatus
						pid, errWait := syscall.Wait4(-1, &status, 0, nil)
						if pid < 0 {
							break
						}
						if errWait != nil {
							log.Error().Err(errWait).Msg("Wait4 failed")
							continue
						}

						allProcs := dbHandle.GetProcs(core.WithAllIfNoFilters)

						procID, procFound := fun.FindKeyBy(allProcs, func(_ core.ProcID, procData core.Proc) bool {
							return procData.Status.Status == core.StatusRunning &&
								procData.Status.Pid == pid
						})
						if !procFound {
							continue
						}

						ebus.Publish(ctx, eventbus.NewPublishProcStopped(procID, status.ExitStatus(), eventbus.EmitReasonDied))
					}
				}
			}()
			return nil
		},
		OnStop: nil,
	})
}

func startStatuser(lc fx.Lifecycle, ebus *eventbus.EventBus, dbHandle db.Handle) {
	// status updater
	statusUpdaterCh := ebus.Subscribe(
		"status_updater",
		eventbus.KindProcStarted,
		eventbus.KindProcStopped,
	)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case event := <-statusUpdaterCh:
						switch e := event.Data.(type) {
						case eventbus.DataProcStarted:
							// TODO: fill/remove cpu, memory
							runningStatus := core.NewStatusRunning(time.Now(), e.Pid, 0, 0)
							if err := dbHandle.SetStatus(e.Proc.ID, runningStatus); err != nil {
								log.Error().
									Uint64("proc_id", e.Proc.ID).
									Any("new_status", runningStatus).
									Msg("set proc status to running")
							}
						case eventbus.DataProcStopped:
							dbStatus := core.NewStatusStopped(e.ExitCode)
							if err := dbHandle.SetStatus(e.ProcID, dbStatus); err != nil {
								if _, ok := xerr.As[db.ProcNotFoundError](err); ok {
									log.Error().
										Uint64("proc_id", e.ProcID).
										Int("exit_code", e.ExitCode).
										Msg("proc not found while trying to set stopped status")
								} else {
									log.Error().
										Uint64("proc_id", e.ProcID).
										Any("new_status", dbStatus).
										Msg("set proc status to stopped")
								}
							}
						}
					}
				}
			}()
			return nil
		},
		OnStop: nil,
	})
}

func startCron(lc fx.Lifecycle, ebus *eventbus.EventBus, dbHandle db.Handle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go cron{
				l:                 log.Logger.With().Str("system", "cron").Logger(),
				db:                dbHandle,
				statusUpdateDelay: 5 * time.Second,
				ebus:              ebus,
			}.start(ctx)
			return nil
		},
		OnStop: nil,
	})
}

func newApp() fx.Option {
	return fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			return fxLogger{Logger: log.Logger.With().Str("component", "fx").Logger()}
		}),
		fx.Provide(readPmConfig),
		fx.Invoke(migrateConfig),
		fx.Provide(func() (db.Handle, error) {
			return db.New(_dirDB)
		}),
		eventbus.Module,
		watcher.Module,
		fx.Invoke(startChildrenStatuser),
		fx.Invoke(startStatuser),
		fx.Invoke(runner.Start),
		fx.Invoke(startCron),
		moduleServer,
	)
}
