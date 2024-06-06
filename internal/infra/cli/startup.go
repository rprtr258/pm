package cli

import (
	"github.com/spf13/cobra"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
)

var _cmdStartup = &cobra.Command{
	Use:    "startup",
	Short:  "run startup processes",
	Args:   cobra.NoArgs,
	Hidden: true,
	RunE: func(*cobra.Command, []string) error {
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

		return errors.Combine(fun.Map[error](func(proc core.Proc) error {
			return app.Start(proc.ID)
		}, procsToStart...)...)
	},
}
