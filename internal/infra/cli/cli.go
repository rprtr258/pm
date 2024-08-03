package cli

import (
	"fmt"
	"os"

	"github.com/rprtr258/scuf"
	"github.com/spf13/cobra"
)

var _app = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "pm",
		Short:        "manage running processes",
		SilenceUsage: true,
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
	// setting template strings
	// 	App.SetUsageFunc(func(cmd *cobra.Command) error {
	// 		scuf.New(os.Stdout).
	// 			NL().
	// 			String(`Usage:
	//   pm COMMAND
	// `).
	// 			Iter(func(yield func(func(scuf.Buffer)) bool) bool {
	// 				for _, group := range cmd.Groups() {
	// 					yield(func(b scuf.Buffer) {
	// 						b.NL().String(group.Title).NL()
	// 						for _, cmd := range cmd.Commands() {
	// 							if cmd.GroupID != group.ID {
	// 								continue
	// 							}
	// 							b.
	// 								String("  ").
	// 								String(cmd.Name(), scuf.FgCyan).
	// 								// String(strings.Join(cmd.Aliases, ", "), scuf.FgGreen).
	// 								String(strings.Repeat(" ", 12-len(cmd.Name()))).
	// 								String(cmd.Short).
	// 								NL()
	// 						}
	// 					})
	// 				}
	// 				return true
	// 			}).
	// 			String(`
	// Additional Commands:
	//   completion  Generate the autocompletion script for the specified shell
	//   help        Help about any command
	//   version     print pm version

	// Flags:
	//   -h, --help   help for pm

	// Use "pm [command] --help" for more information about a command.`)
	// 		return nil
	// 	})
	_app.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		scuf.New(os.Stdout).
			String(cmd.Short, scuf.FgBlue).
			String(fmt.Sprintf(`

%s`,
				cmd.UsageString()))
	})

	_app.SetArgs(argv[1:])
	return _app.Execute()
}
