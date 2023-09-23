package daemon

import (
	"context"
	"path/filepath"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

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

type Server struct {
	db               db.Handle
	ebus             *eventbus.EventBus
	homeDir, logsDir string
}

func newServer(lc fx.Lifecycle, ebus *eventbus.EventBus, dbHandle db.Handle) *Server {
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
