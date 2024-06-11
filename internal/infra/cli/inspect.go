package cli

import (
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
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
StderrFile: {{.StderrFile}}{{if .Watch.Valid}}
Watch: {{.Watch.Value}}{{end}}
Status:
	Status: {{.Status}}{{if eq (print .Status) "running"}}
	StartTime: {{formatTime .StartTime}}
	CPU: {{.CPU}}
	Memory: {{.Memory}}{{end}}
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

			filterFunc := core.FilterFunc(
				core.WithAllIfNoFilters,
				core.WithGeneric(args...),
				core.WithIDs(ids...),
				core.WithNames(names...),
				core.WithTags(tags...),
			)
			procsToShow := appp.
				List().
				Filter(func(ps core.ProcStat) bool {
					return filterFunc(ps.Proc)
				}).
				ToSlice()

			for _, proc := range procsToShow {
				if err := _procInspectTemplate.Execute(os.Stdout, proc); err != nil {
					log.Error().Err(err).Msg("render inspect template")
				}

				if proc.Status == core.StatusRunning || proc.Status == core.StatusCreated {
					fmt.Println("\tSHIM_PID:", proc.ShimPID)
				}
				if proc.Status == core.StatusRunning {
					fmt.Println("\tPID:", proc.ChildPID.Value)
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
