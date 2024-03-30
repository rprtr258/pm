package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
)

var _cmdVersion = &cobra.Command{
	Use:   "version",
	Short: "print pm version",
	Args:  cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		fmt.Println(core.Version)
		return nil
	},
}
