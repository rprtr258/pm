package cli

import (
	"errors"
	"fmt"

	"github.com/rprtr258/xerr"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
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
			slog.Info("current version is older, run `pm daemon restart` to update",
				"curVersion", config.Version,
			)
		case 0:
		case 1:
			slog.Warn("current version is newer, please update pm", "curVersion", config.Version)
		default:
			return xerr.NewM("invalid version compare result", xerr.Fields{"cmp": cmp})
		}

		return nil
	},
}
