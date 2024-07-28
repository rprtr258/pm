package cli

import (
	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
)

var _cmdStartup = &cobra.Command{
	Use:    "startup",
	Short:  "run startup processes",
	Args:   cobra.NoArgs,
	Hidden: true,
	RunE: func(*cobra.Command, []string) error {
		procsToStart := listProcs(dbb).
			Filter(func(p core.ProcStat) bool {
				return p.Startup && p.Status != core.StatusRunning
			}).
			ToSlice()

		return errors.Combine(fun.Map[error](func(proc core.ProcStat) error {
			return implStart(dbb, proc.ID)
		}, procsToStart...)...)
	},
}
