package cli

import (
	"fmt"
	"time"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
	"github.com/spf13/cobra"
)

var _cmdInspect = func() *cobra.Command {
	var names, ids, tags []string
	cmd := &cobra.Command{
		Use:               "inspect [name|tag|id]...",
		Short:             "inspect process",
		Aliases:           []string{"i"},
		GroupID:           "inspection",
		ValidArgsFunction: completeArgGenericSelector,
		RunE: func(_ *cobra.Command, args []string) error {
			appp, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrap(errNewApp, "new app")
			}

			procsToShow := appp.
				List().
				Filter(core.FilterFunc(
					core.WithAllIfNoFilters,
					core.WithGeneric(args...),
					core.WithIDs(ids...),
					core.WithNames(names...),
					core.WithTags(tags...),
				)).
				ToSlice()

			procs := linuxprocess.List()

			for _, proc := range procsToShow {
				fmt.Printf(`ID: %s
Name: %s
Tags: %v
Command: %s
Args: %v
Cwd: %s
Env: %v
StdoutFile: %s
StderrFile: %s
Watch: %v
Status:
    Type: %s
    StartTime: %s
    CPU: %d
    Memory: %d
    ExitCode: %d
`,
					proc.ID,
					proc.Name,
					proc.Tags,
					proc.Command,
					proc.Args,
					proc.Cwd,
					proc.Env,
					proc.StdoutFile,
					proc.StderrFile,
					proc.Watch,
					proc.Status.Status.String(),
					proc.Status.StartTime.Format(time.DateTime),
					proc.Status.CPU,
					proc.Status.Memory,
					proc.Status.ExitCode,
				)

				var agentPid int
				for _, p := range procs {
					if p.Environ[app.EnvPMID] == string(proc.ID) {
						agentPid = p.Handle.Pid
						fmt.Println("AGENT_PID:", agentPid)
						break
					}
				}
				if agentPid != 0 {
					for _, p := range procs {
						if stat, err := linuxprocess.ReadProcessStat(p.Handle.Pid); err == nil {
							if stat.Ppid == agentPid {
								agentPid = p.Handle.Pid
								fmt.Println("PID:", agentPid)
								fmt.Println("PROCESS_ENV:")
								for k, v := range p.Environ {
									fmt.Printf("    %s: %q\n", k, v)
								}
								break
							}
						}
					}
				}
			}

			return nil
		},
	}
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	return cmd
}()
