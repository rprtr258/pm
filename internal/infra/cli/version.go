package cli

import (
	"errors"
	"fmt"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/semver"

	"github.com/rprtr258/pm/internal/core"
)

var _versionCmd = &cli.Command{
	Name:    "version",
	Aliases: []string{"v"},
	Usage:   "print pm version",
	Action: func(c *cli.Context) error {
		fmt.Println(core.Version)

		config, errRead := core.ReadConfig()
		if errRead != nil {
			if errors.Is(errRead, core.ErrConfigNotExists) {
				return nil
			}

			return xerr.NewWM(errRead, "read config")
		}

		switch cmp := semver.Compare(config.Version, core.Version); cmp {
		case -1:
			log.Info().
				Str("curVersion", config.Version).
				Msg("current version is older, run `pm daemon restart` to update")
		case 0:
		case 1:
			log.Warn().
				Str("curVersion", config.Version).
				Msg("current version is newer, please update pm")
		}

		return nil
	},
}
