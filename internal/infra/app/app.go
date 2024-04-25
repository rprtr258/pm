package app

import (
	"fmt"
	"path/filepath"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/errors"
)

var (
	_dirProcsLogs = filepath.Join(core.DirHome, "logs")
	_dirDB        = filepath.Join(core.DirHome, "db")
)

func ReadPMConfig() (core.Config, error) {
	config, errRead := core.ReadConfig()
	if errRead != nil {
		if errRead != core.ErrConfigNotExists {
			return fun.Zero[core.Config](), errors.Wrapf(errRead, "read config for migrate")
		}

		log.Info().Msg("writing initial config...")

		if errWrite := core.WriteConfig(core.DefaultConfig); errWrite != nil {
			return fun.Zero[core.Config](), errors.Wrapf(errWrite, "write initial config")
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
		return errors.Wrapf(errWrite, "write config for migrate, version=%s", core.Version)
	}

	return nil
}

type App struct {
	DB              db.Handle
	DirHome, DirLos string
	Config          core.Config
}

func New() (App, error) {
	cfg, errCfg := ReadPMConfig()
	if errCfg != nil {
		return App{}, fmt.Errorf("config: %w", errCfg)
	}

	if errMigrate := MigrateConfig(cfg); errMigrate != nil {
		return fun.Zero[App](), fmt.Errorf("migrate: %w", errMigrate)
	}

	dbHandle, errDB := db.New(_dirDB)
	if errDB != nil {
		return fun.Zero[App](), errors.Wrapf(errDB, "new db, dir=%s", _dirDB)
	}

	config, errConfig := core.ReadConfig()
	if errConfig != nil {
		if errConfig == core.ErrConfigNotExists {
			return App{
				DB:      dbHandle,
				DirHome: core.DirHome,
				DirLos:  _dirProcsLogs,
				Config:  core.DefaultConfig,
			}, nil
		}

		return fun.Zero[App](), errors.Wrapf(errConfig, "read app config")
	}

	return App{
		DB:      dbHandle,
		DirHome: core.DirHome,
		DirLos:  _dirProcsLogs,
		Config:  config,
	}, nil
}