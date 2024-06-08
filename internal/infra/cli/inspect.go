package cli

import (
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

var _procInspectTemplate = template.Must(template.New("proc").
	Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format(time.DateTime)
		},
	}).
	Parse(`ID: {{.ID}}
Name: {{.Name}}
Tags: {{.Tags}}
Command: {{.Command}}
Args: {{.Args}}
Cwd: {{.Cwd}}
Env: {{.Env}}
StdoutFile: {{.StdoutFile}}
StderrFile: {{.StderrFile}}
Watch: {{.Watch}}
Status:
  Status: {{.Status.Status}}{{if eq (print .Status.Status) "running"}}
  StartTime: {{formatTime .Status.StartTime}}
  CPU: {{.Status.CPU}}
  Memory: {{.Status.Memory}}{{else}}
  ExitCode: {{.Status.ExitCode}}{{end}}
`))

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
				return errors.Wrapf(errNewApp, "new app")
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
				_ = _procInspectTemplate.Execute(os.Stdout, proc)

				var shimPid int
				for _, p := range procs {
					if p.Environ[app.EnvPMID] == string(proc.ID) {
						shimPid = p.Handle.Pid
						fmt.Println("SHIM_PID:", shimPid)
						break
					}
				}
				if shimPid != 0 {
					for _, p := range procs {
						if stat, err := linuxprocess.ReadProcessStat(p.Handle.Pid); err == nil {
							if stat.Ppid == shimPid {
								shimPid = p.Handle.Pid
								fmt.Println("PID:", shimPid)
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
