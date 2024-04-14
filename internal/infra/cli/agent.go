package cli

import (
	"encoding/json"
	"time"

	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
)

var _cmdAgent = &cobra.Command{
	Use:    "agent",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(_ *cobra.Command, args []string) error {
		var config core.Proc
		if err := json.Unmarshal([]byte(args[0]), &config); err != nil {
			return errors.Wrapf(err, "unmarshal agent config: %s", args[0])
		}

		// TODO: remove
		// a little sleep to wait while calling process closes db file
		time.Sleep(1 * time.Second)

		app, errNewApp := app.New()
		if errNewApp != nil {
			return errors.Wrapf(errNewApp, "new app")
		}

		if err := app.StartRaw(config); err != nil {
			return errors.Wrapf(err, "run: %v", config)
		}

		return nil
	},
}
