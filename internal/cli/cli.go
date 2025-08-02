package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func addGroup(
	cmd *cobra.Command,
	title string,
	cmds ...*cobra.Command,
) {
	id := strings.ToLower(title)
	cmd.AddGroup(&cobra.Group{
		ID:    id,
		Title: title + ":",
	})
	for _, c := range cmds {
		cmd.AddCommand(c)
		c.GroupID = id
	}
}

var _app = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "pm",
		Short:         "manage running processes",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(_cmdVersion)
	cmd.AddCommand(_cmdStartup)
	cmd.AddCommand(_cmdShim)
	cmd.AddCommand(_cmdRunStartup)
	cmd.AddCommand(_cmdTUI)
	addGroup(cmd, "Inspection",
		_cmdList,
		_cmdLogs,
		_cmdInspect,
	)
	addGroup(cmd, "Management",
		_cmdRun,
		_cmdStart,
		_cmdRestart,
		_cmdStop,
		_cmdDelete,
		_cmdSignal,
		_cmdAttach,
	)
	return cmd
}()

func Run(argv []string) error {
	_app.SetArgs(argv[1:])
	return _app.Execute()
}
