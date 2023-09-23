package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/runner"
	"github.com/rprtr258/pm/internal/core/daemon/watcher"
	"github.com/rprtr258/pm/internal/core/fx"
	"github.com/rprtr258/pm/internal/infra/db"
)

var (
	_dirProcsLogs = filepath.Join(core.DirHome, "logs")
	_filePid      = filepath.Join(core.DirHome, "pm.pid")
	_fileLog      = filepath.Join(core.DirHome, "pm.log")
	_dirDB        = filepath.Join(core.DirHome, "db")
)

func readPmConfig() (core.Config, error) {
	config, errRead := core.ReadConfig()
	if errRead != nil {
		if errRead != core.ErrConfigNotExists {
			return fun.Zero[core.Config](), xerr.NewWM(errRead, "read config for migrate")
		}

		log.Info().Msg("writing initial config...")

		if errWrite := core.WriteConfig(core.DefaultConfig); errWrite != nil {
			return fun.Zero[core.Config](), xerr.NewWM(errWrite, "write initial config")
		}

		return core.DefaultConfig, nil
	}

	return config, nil
}

func migrateConfig(config core.Config) error {
	if config.Version == core.Version {
		return nil
	}

	config.Version = core.Version
	if errWrite := core.WriteConfig(config); errWrite != nil {
		return xerr.NewWM(errWrite, "write config for migrate", xerr.Fields{"version": core.Version})
	}

	return nil
}

// TODO: not working, fix
func startChildrenStatuser(ebus *eventbus.EventBus, dbHandle db.Handle) fx.Lifecycle {
	c := make(chan os.Signal, 10) // arbitrary buffer size
	signal.Notify(c, syscall.SIGCHLD)
	return fx.Lifecycle{
		Name:  "children-statuser",
		Start: nil,
		StartAsync: func(ctx context.Context) {
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

					log.Info().Int("pid", pid).Msg("child died")

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
		},
		Close: nil,
	}
}

func startStatuser(ebus *eventbus.EventBus, dbHandle db.Handle) fx.Lifecycle {
	// status updater
	statusUpdaterCh := ebus.Subscribe(
		"status_updater",
		eventbus.KindProcStarted,
		eventbus.KindProcStopped,
	)
	return fx.Lifecycle{
		StartAsync: func(ctx context.Context) {
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
		},
	}
}

func NewApp() (*Server, fx.Lifecycle, error) {
	cfg, errCfg := readPmConfig()
	if errCfg != nil {
		return nil, fun.Zero[fx.Lifecycle](), fmt.Errorf("config: %w", errCfg)
	}

	if errMigrate := migrateConfig(cfg); errMigrate != nil {
		return nil, fun.Zero[fx.Lifecycle](), fmt.Errorf("migrate: %w", errMigrate)
	}

	dbHandle, errDB := db.New(_dirDB)
	if errDB != nil {
		return nil, fun.Zero[fx.Lifecycle](), fmt.Errorf("db: %w", errDB)
	}

	ebus, ebusLc := eventbus.Module(dbHandle)

	watcherLc, errWatcher := watcher.Module(ebus)
	if errWatcher != nil {
		return nil, fun.Zero[fx.Lifecycle](), fmt.Errorf("watcher: %w", errWatcher)
	}

	return newServer(ebus, dbHandle), fx.Combine("app",
		ebusLc,
		watcherLc,
		startChildrenStatuser(ebus, dbHandle),
		startStatuser(ebus, dbHandle),
		runner.Start(ebus, dbHandle),
		startCron(ebus, dbHandle),
	), nil
}

type Server struct {
	db               db.Handle
	ebus             *eventbus.EventBus
	homeDir, logsDir string
}

func newServer(ebus *eventbus.EventBus, dbHandle db.Handle) *Server {
	return &Server{
		db:      dbHandle,
		ebus:    ebus,
		homeDir: core.DirHome,
		logsDir: _dirProcsLogs,
	}
}

// Start - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (srv *Server) Start(ctx context.Context, id core.ProcID) {
	srv.ebus.Publish(ctx, eventbus.NewPublishProcStartRequest(id, eventbus.EmitReasonByUser))
}

func (s *Server) Stop(ctx context.Context, id core.ProcID) {
	s.ebus.Publish(ctx, eventbus.NewPublishProcStopRequest(id, eventbus.EmitReasonByUser))
}

func (s *Server) List(_ context.Context) map[core.ProcID]core.Proc {
	return s.db.GetProcs(core.WithAllIfNoFilters)
}

// Signal - send signal to process
func (s *Server) Signal(ctx context.Context, id core.ProcID, signal syscall.Signal) {
	s.ebus.Publish(ctx, eventbus.NewPublishProcSignalRequest(signal, id))
}
