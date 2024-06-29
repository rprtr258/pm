package app

import (
	"path/filepath"

	"github.com/rprtr258/fun"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/errors"
)

const EnvPMID = "PM_PMID"

var _dirDB = filepath.Join(core.DirHome, "db")

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

func New() (db.Handle, core.Config, error) {
	config, errConfig := core.ReadConfig()
	if errConfig != nil {
		return fun.Zero[db.Handle](), fun.Zero[core.Config](), errors.Wrap(errConfig, "config")
	}

	if errMigrate := MigrateConfig(config); errMigrate != nil {
		return fun.Zero[db.Handle](), fun.Zero[core.Config](), errors.Wrap(errMigrate, "migrate")
	}

	dbFs, errDB := db.InitRealDir(_dirDB)
	if errDB != nil {
		return fun.Zero[db.Handle](), fun.Zero[core.Config](), errors.Wrapf(errDB, "new db, dir=%q", _dirDB)
	}

	return db.New(dbFs), config, nil
}
