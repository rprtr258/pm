package cli

import "github.com/spf13/cobra"

var _app = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "pm",
		Short:         "manage running processes",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(_cmdVersion)
	cmd.AddCommand(_cmdShim)
	cmd.AddCommand(_cmdStartup)

	cmd.AddGroup(&cobra.Group{ID: "inspection", Title: "Inspection:"})
	cmd.AddCommand(_cmdList)
	cmd.AddCommand(_cmdLogs)
	cmd.AddCommand(_cmdInspect)

	cmd.AddGroup(&cobra.Group{ID: "management", Title: "Management:"})
	cmd.AddCommand(_cmdRun)
	cmd.AddCommand(_cmdStart)
	cmd.AddCommand(_cmdRestart)
	cmd.AddCommand(_cmdStop)
	cmd.AddCommand(_cmdDelete)
	cmd.AddCommand(_cmdSignal)
	return cmd
}()

func Run(argv []string) error {
	_app.SetArgs(argv[1:])
	return _app.Execute()
}
