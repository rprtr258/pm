package cli

import (
	"encoding/json"
	"time"

	"github.com/rprtr258/xerr"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/daemon"
)

var _cmdAgent = &cli.Command{
	Name:   daemon.CmdAgent,
	Hidden: true,
	Action: func(ctx *cli.Context) error {
		arg := core.PMID(ctx.Args().First())
		if arg == "" {
			return xerr.NewM("Usage: pm agent <config>")
		}

		// TODO: remove
		// a little sleep to wait while calling process closes db file
		time.Sleep(1 * time.Second)

		app, errNewApp := daemon.New()
		if errNewApp != nil {
			return xerr.NewWM(errNewApp, "new app")
		}

		var proc core.Proc
		if err := json.Unmarshal([]byte(arg), &proc); err != nil {
			return xerr.NewWM(err, "unmarshal config", xerr.Fields{"arg": arg})
		}

		if err := app.StartRaw(proc); err != nil {
			return xerr.NewWM(err, "run", xerr.Fields{"arg": arg})
		}

		return nil
	},
}
