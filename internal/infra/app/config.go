package app

import (
	stdErrors "errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/errors"
)

const EnvPMID = "PM_PMID"

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

func ensureDir(dirname string) error {
	if _, errStat := os.Stat(dirname); errStat == nil {
		return nil
	} else if !stdErrors.Is(errStat, fs.ErrNotExist) {
		return errors.Wrapf(errStat, "stat dir")
	}

	log.Info().Str("dir", dirname).Msg("creating dir...")
	if errMkdir := os.Mkdir(dirname, 0o755); errMkdir != nil {
		return errors.Wrapf(errMkdir, "create dir")
	}

	return nil
}

func New() (db.Handle, core.Config, error) {
	config, errConfig := core.ReadConfig()
	if errConfig != nil {
		return fun.Zero[db.Handle](), fun.Zero[core.Config](), errors.Wrap(errConfig, "config")
	}

	if err := func() error {
		if errMigrate := MigrateConfig(config); errMigrate != nil {
			return errors.Wrap(errMigrate, "migrate")
		}

		// // TODO: uncomment
		// if _, errMigrate := db.Migrate(config.DirDB, config.Version, core.Version); errMigrate != nil {
		// 	retu errors.Wrap(errMigrate, "migrate")
		// }

		if err := ensureDir(config.DirHome); err != nil {
			return errors.Wrapf(err, "ensure home dir %s", config.DirHome)
		}

		_dirProcsLogs := filepath.Join(config.DirHome, "logs")
		if err := ensureDir(_dirProcsLogs); err != nil {
			return errors.Wrapf(err, "ensure logs dir %s", _dirProcsLogs)
		}

		return nil
	}(); err != nil {
		return fun.Zero[db.Handle](), fun.Zero[core.Config](), err
	}

	dbFs, errDB := db.InitRealDir(config.DirDB)
	if errDB != nil {
		return fun.Zero[db.Handle](), fun.Zero[core.Config](), errors.Wrapf(errDB, "new db, dir=%q", config.DirDB)
	}

	return db.New(dbFs), config, nil
}
