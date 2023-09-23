package daemon

import (
	"context"
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
	"github.com/rprtr258/pm/internal/infra/db"
)

var (
	_dirProcsLogs = filepath.Join(core.DirHome, "logs")
	_filePid      = filepath.Join(core.DirHome, "pm.pid")
	_fileLog      = filepath.Join(core.DirHome, "pm.log")
	_dirDB        = filepath.Join(core.DirHome, "db")
)

func ReadPMConfig() (core.Config, error) {
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

func MigrateConfig(config core.Config) error {
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
func StartChildrenStatuser(ctx context.Context, ebus *eventbus.EventBus, dbHandle db.Handle) {
	c := make(chan os.Signal, 10) // arbitrary buffer size
	signal.Notify(c, syscall.SIGCHLD)

	for sig := range c {
		log.Debug().Any("sig", sig).Msg("received SIGCHLD")
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
}

func StartStatuser(ctx context.Context, ebus *eventbus.EventBus, dbHandle db.Handle) {
	// status updater
	statusUpdaterQ := ebus.Subscribe(
		"status_updater",
		eventbus.KindProcStarted,
		eventbus.KindProcStopped,
	)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			event, ok := statusUpdaterQ.Pop()
			if !ok {
				continue
			}

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
}

type Server struct {
	db               db.Handle
	ebus             *eventbus.EventBus
	homeDir, logsDir string
}

func NewServer(ebus *eventbus.EventBus, dbHandle db.Handle) *Server {
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
