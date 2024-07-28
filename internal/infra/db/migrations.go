package db

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/mod/semver"

	"github.com/rprtr258/pm/internal/infra/errors"
)

type migration struct {
	do      func(dirDB string) error
	version string
}

var Migrations = []migration{
	{ // initial version
		version: "0.0.1",
		do: func(dirDB string) error {
			if _, errDB := InitRealDir(dirDB); errDB != nil {
				return errors.Wrapf(errDB, "new db, dir=%s", dirDB)
			}

			return nil
		},
	},
}

// TODO: unused
func Migrate(
	dirDB string,
	fromVersion, toVersion string,
) (string, error) {
	lastVersion := fromVersion
	for _, m := range Migrations {
		if semver.Compare(fromVersion, m.version) == -1 &&
			semver.Compare(m.version, toVersion) == -1 {
			log.Info().
				Str("from", lastVersion).
				Str("to", m.version).
				Msg("migrating...")

			if err := m.do(dirDB); err != nil {
				return lastVersion, errors.Wrapf(err, "migrate from=%s to=%s", lastVersion, m.version)
			}

			lastVersion = m.version
		}
	}

	return lastVersion, nil
}
