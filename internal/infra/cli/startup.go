package cli

import (
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
)

var _cmdStartup = &cobra.Command{
	Use:    "startup",
	Short:  "run startup processes",
	Hidden: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		app, errNewApp := app.New()
		if errNewApp != nil {
			return errors.Wrapf(errNewApp, "new app")
		}

		procsToStart := app.
			List().
			Filter(func(p core.Proc) bool {
				return p.Startup && p.Status.Status != core.StatusRunning
			}).
			ToSlice()

		var merr error
		for _, proc := range procsToStart {
			multierr.AppendInto(&merr, app.Start(proc.ID))
		}
		return merr
	},
}
