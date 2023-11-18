package daemon

import (
	"context"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/watcher"
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

func StatusSetStarted(dbHandle db.Handle, id core.PMID) {
	// TODO: fill/remove cpu, memory
	runningStatus := core.NewStatusRunning(time.Now(), 0, 0)
	if err := dbHandle.SetStatus(id, runningStatus); err != nil {
		log.Error().
			Stringer("pmid", id).
			Any("new_status", runningStatus).
			Msg("set proc status to running")
	}
}

func StatusSetStopped(dbHandle db.Handle, id core.PMID) {
	dbStatus := core.NewStatusStopped()
	if err := dbHandle.SetStatus(id, dbStatus); err != nil {
		if _, ok := xerr.As[db.ProcNotFoundError](err); ok {
			log.Error().
				Stringer("pmid", id).
				Msg("proc not found while trying to set stopped status")
		} else {
			log.Error().
				Stringer("pmid", id).
				Any("new_status", dbStatus).
				Msg("set proc status to stopped")
		}
	}
}

type Server struct {
	db               db.Handle
	ebus             *eventbus.EventBus
	W                watcher.Watcher
	homeDir, logsDir string
}

func NewServer(ebus *eventbus.EventBus, dbHandle db.Handle, w watcher.Watcher) *Server {
	return &Server{
		db:      dbHandle,
		ebus:    ebus,
		homeDir: core.DirHome,
		logsDir: _dirProcsLogs,
		W:       w,
	}
}

// Start - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (s *Server) Start(ctx context.Context, id core.PMID) {
	s.ebus.Publish(ctx, eventbus.NewPublishProcStartRequest(id, eventbus.EmitReasonByUser))
}

func (s *Server) Stop(ctx context.Context, id core.PMID) {
	s.ebus.Publish(ctx, eventbus.NewPublishProcStopRequest(id, eventbus.EmitReasonByUser))
}

func (s *Server) List(_ context.Context) map[core.PMID]core.Proc {
	procs := s.db.GetProcs(core.WithAllIfNoFilters)
	for id, proc := range procs {
		if proc.Status.Status != core.StatusRunning {
			continue
		}

		// TODO: uncomment
		// if _, err := linuxprocess.ReadProcessStat(proc.PMID); err != nil {
		// 	proc.Status = core.NewStatusStopped()
		// 	if errSet := s.db.SetStatus(id, proc.Status); errSet != nil {
		// 		log.Error().Err(errSet).Msg("failed to update status to stopped")
		// 	}
		// }
		procs[id] = proc
	}
	return procs
}

// Signal - send signal to process
func (s *Server) Signal(ctx context.Context, id core.PMID, signal syscall.Signal) {
	s.ebus.Publish(ctx, eventbus.NewPublishProcSignalRequest(signal, id))
}
