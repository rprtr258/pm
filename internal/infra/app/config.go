package app

import (
	stdErrors "errors"
	"io/fs"
	"os"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

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
	var (
		config core.Config
		dbFs   afero.Fs
	)
	if err := func() error {
		dirHome := core.DirHome()

		if err := ensureDir(dirHome); err != nil {
			return errors.Wrapf(err, "ensure home dir %s", dirHome)
		}

		var errConfig error
		config, errConfig = core.ReadConfig()
		if errConfig != nil {
			return errors.Wrap(errConfig, "config")
		}

		if err := ensureDir(config.DirLogs); err != nil {
			return errors.Wrapf(err, "ensure logs dir %s", config.DirLogs)
		}

		if errMigrate := MigrateConfig(config); errMigrate != nil {
			return errors.Wrap(errMigrate, "migrate")
		}

		// // TODO: uncomment
		// if _, errMigrate := db.Migrate(config.DirDB, config.Version, core.Version); errMigrate != nil {
		// 	retu errors.Wrap(errMigrate, "migrate")
		// }

		var errDB error
		dbFs, errDB = db.InitRealDir(config.DirDB)
		if errDB != nil {
			return errors.Wrapf(errDB, "new db, dir=%q", config.DirDB)
		}

		// TODO: cleanup logs files which are not bound to any existing process (in any status)

		return nil
	}(); err != nil {
		return fun.Zero[db.Handle](), fun.Zero[core.Config](), err
	}

	return db.New(dbFs), config, nil
}
