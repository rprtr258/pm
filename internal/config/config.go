package config

import (
	stdErrors "errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/pm/internal/errors"
)

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

func pruneLogs(db db.Handle) error {
	logFiles, err := os.ReadDir(core.DirLogs)
	if err != nil {
		return errors.Wrapf(err, "read log dir %s", core.DirLogs)
	}

	procs, err := db.List(core.WithAllIfNoFilters)
	if err != nil {
		return errors.Wrapf(err, "get procs")
	}

	ids := make(map[core.PMID]struct{}, len(procs))
	for id := range procs {
		ids[id] = struct{}{}
	}

	for _, logFile := range logFiles {
		if len(logFile.Name()) >= core.PMIDLen {
			id := logFile.Name()[:core.PMIDLen]
			if _, ok := ids[core.PMID(id)]; ok {
				continue
			}
		}

		// proc not found, remove log file
		filename := filepath.Join(core.DirLogs, logFile.Name())
		log.Debug().
			Str("file", filename).
			Msg("pruning log file")
		if errRemove := os.Remove(filename); errRemove != nil {
			return errors.Wrapf(errRemove, "remove log file %s", logFile.Name())
		}
	}

	return nil
}

func setupLogger(config core.Config) {
	level := fun.IF(config.Debug, zerolog.DebugLevel, zerolog.InfoLevel)

	log.Logger = zerolog.New(os.Stderr).
		Level(level).
		Output(zerolog.ConsoleWriter{ //nolint:exhaustruct // not needed
			Out: os.Stderr,
			FormatLevel: func(i any) string {
				s, _ := i.(string)
				bg := fun.Switch(s, scuf.BgRed).
					Case(scuf.BgBlue, zerolog.LevelInfoValue).
					Case(scuf.BgGreen, zerolog.LevelWarnValue).
					Case(scuf.BgYellow, zerolog.LevelErrorValue).
					End()

				return scuf.String(" "+strings.ToUpper(s)+" ", bg, scuf.FgBlack)
			},
			FormatTimestamp: func(i any) string {
				s, _ := i.(string)
				t, err := time.Parse(zerolog.TimeFieldFormat, s)
				if err != nil {
					return s
				}

				return scuf.String(t.Format("[15:06:05]"), scuf.ModFaint, scuf.FgWhite)
			},
		})
}

func New() (db.Handle, core.Config, error) {
	setupLogger(core.DefaultConfig)

	var (
		config core.Config
		dbFs   afero.Fs
	)
	if err := func() error {
		if err := ensureDir(core.DirHome); err != nil {
			return errors.Wrapf(err, "ensure home dir %s", core.DirHome)
		}

		var errConfig error
		config, errConfig = core.ReadConfig()
		if errConfig != nil {
			return errors.Wrap(errConfig, "config")
		}

		setupLogger(config)

		if err := ensureDir(core.DirLogs); err != nil {
			return errors.Wrapf(err, "ensure logs dir %s", core.DirLogs)
		}

		if core.Version != "dev" && config.Version != core.Version {
			return errors.Newf("config version mismatch, config=%s, pm=%s", config.Version, core.Version)
		}

		var errDB error
		dbFs, errDB = db.InitRealDir(core.DirDB)
		if errDB != nil {
			return errors.Wrapf(errDB, "new db, dir=%q", core.DirDB)
		}

		if err := pruneLogs(db.New(dbFs)); err != nil {
			return errors.Wrap(err, "prune logs")
		}

		return nil
	}(); err != nil {
		return fun.Zero[db.Handle](), fun.Zero[core.Config](), err
	}

	return db.New(dbFs), config, nil
}
