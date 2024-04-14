package db

import (
	"path/filepath"

	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/mod/semver"

	"github.com/rprtr258/pm/internal/core"
)

type migration struct {
	do      func() error
	version string
}

var Migrations = []migration{
	{ // initial version
		version: "0.0.1",
		do: func() error {
			if _, err := New(filepath.Join(core.DirHome, "db")); err != nil {
				return errors.Wrapf(err, "create db handler")
			}

			return nil
		},
	},
}

func Migrate(fromVersion, toVersion string) (string, error) {
	lastVersion := fromVersion
	for _, m := range Migrations {
		if semver.Compare(fromVersion, m.version) == -1 &&
			semver.Compare(m.version, toVersion) == -1 {
			log.Info().
				Str("from", lastVersion).
				Str("to", m.version).
				Msg("migrating...")

			if err := m.do(); err != nil {
				return lastVersion, errors.Wrapf(err, "migrate", map[string]any{
					"from": lastVersion,
					"to":   m.version,
				})
			}

			lastVersion = m.version
		}
	}

	return lastVersion, nil
}
