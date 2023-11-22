package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
)

var (
	_dirProcsLogs = filepath.Join(core.DirHome, "logs")
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

type App struct {
	db               db.Handle
	homeDir, logsDir string
	config           core.Config
}

func New() (App, error) {
	log.Logger = zerolog.New(os.Stderr).With().
		Timestamp().
		Caller().
		Logger()

	cfg, errCfg := ReadPMConfig()
	if errCfg != nil {
		return App{}, fmt.Errorf("config: %w", errCfg)
	}

	if errMigrate := MigrateConfig(cfg); errMigrate != nil {
		return App{}, fmt.Errorf("migrate: %w", errMigrate)
	}

	dbHandle, errDB := db.New(_dirDB)
	if errDB != nil {
		return App{}, xerr.NewWM(errDB, "new db", xerr.Fields{"dir": _dirDB})
	}

	// TODO: move to agent
	// watcher := watcher.New(ebus)
	// go watcher.Start(ctx)

	config, errConfig := core.ReadConfig()
	if errConfig != nil {
		if errConfig == core.ErrConfigNotExists {
			return App{
				db:      dbHandle,
				homeDir: core.DirHome,
				logsDir: _dirProcsLogs,
				config:  core.DefaultConfig,
			}, nil
		}

		return fun.Zero[App](), xerr.NewWM(errConfig, "read app config")
	}

	return App{
		db:      dbHandle,
		homeDir: core.DirHome,
		logsDir: _dirProcsLogs,
		config:  config,
	}, nil
}
