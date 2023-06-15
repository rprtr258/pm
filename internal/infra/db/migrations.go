package db

import (
	"path/filepath"

	"github.com/rprtr258/log"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/xerr"
	"golang.org/x/mod/semver"
)

type migration struct {
	do      func() error
	version string
}

var Migrations = []migration{
	{ // initial version
		version: "0.0.1",
		do: func() error {
			handler, err := New(filepath.Join(core.DirHome, "db"))
			if err != nil {
				return xerr.NewWM(err, "create db handler")
			}

			errFlush := handler.procs.Flush()
			if errFlush != nil {
				return xerr.NewWM(errFlush, "flush procs")
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
			log.Infof("migrating...", log.F{
				"from": lastVersion,
				"to":   m.version,
			})

			if err := m.do(); err != nil {
				return lastVersion, xerr.NewWM(err, "migrate", xerr.Fields{
					"from": lastVersion,
					"to":   m.version,
				})
			}

			lastVersion = m.version
		}
	}

	return lastVersion, nil
}
