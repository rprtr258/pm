package cli

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/aquasecurity/table"
	"github.com/kballard/go-shellquote"
	cmp2 "github.com/rprtr258/cmp"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/scuf"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

const (
	_formatTable   = "table"
	_formatCompact = "compact"
	_formatJSON    = "json"
	_formatShort   = "short"
)

var _formats = []string{
	_formatTable,
	_formatCompact,
	_formatJSON,
	_formatShort,
}

const _shortIDLength = 8

func mapStatus(status core.Status) (string, time.Duration) {
	switch status.Status {
	case core.StatusCreated:
		return scuf.String("created", scuf.FgYellow), 0
	case core.StatusRunning:
		// TODO: get back real pid
		return scuf.String("running", scuf.FgGreen), time.Since(status.StartTime)
	case core.StatusStopped:
		return scuf.String(fmt.Sprintf("stopped(%d)", status.ExitCode), scuf.FgRed), 0
	case core.StatusInvalid:
		return scuf.String(fmt.Sprintf("invalid(%#v)", status.Status), scuf.FgRed), 0
	default:
		return scuf.String(fmt.Sprintf("BROKEN(%T)", status), scuf.FgRed), 0
	}
}

func renderTable(procs []core.Proc, setRowLines bool) {
	procsTable := table.New(os.Stdout)
	procsTable.SetDividers(table.UnicodeRoundedDividers)
	procsTable.SetHeaders("id", "name", "status", "uptime", "tags" /*"cpu", "memory",*/, "cmd")
	procsTable.SetHeaderStyle(table.StyleBold)
	procsTable.SetLineStyle(table.StyleDim)
	procsTable.SetRowLines(setRowLines)
	for _, proc := range procs {
		// TODO: if errored/stopped show time since start instead of uptime (not in place of)
		status, uptime := mapStatus(proc.Status)

		procsTable.AddRow(
			scuf.String(proc.ID.String()[:_shortIDLength], scuf.FgCyan, scuf.ModBold),
			proc.Name,
			status,
			fun.
				If(proc.Status.Status != core.StatusRunning, "").
				Else(uptime.Truncate(time.Second).String()),
			strings.Join(proc.Tags, "\n"),
			// fmt.Sprint(proc.Status.CPU),
			// fmt.Sprint(proc.Status.Memory),
			shellquote.Join(append([]string{proc.Command}, proc.Args...)...),
		)
	}
	procsTable.Render()
}

var _usageFlagSort = scuf.NewString(func(b scuf.Buffer) {
	b.
		String("Sort order. Available sort fields: ").
		String("id", scuf.FgYellow).String(", ").
		String("name", scuf.FgYellow).String(", ").
		String("status", scuf.FgYellow).String(", ").
		String("uptime", scuf.FgYellow).
		String(". Order can be changed by adding ").
		String(":asc", scuf.FgRed).
		String(" or ").
		String(":desc", scuf.FgRed)
})

func unmarshalFlagSort(value string) (func(a, b core.Proc) int, error) {
	sortField := value
	sortOrder := "asc"
	if i := strings.IndexRune(sortField, ':'); i != -1 {
		sortField, sortOrder = sortField[:i], sortField[i+1:]
	}

	var less func(a, b core.Proc) int
	switch sortField {
	case "id":
		less = func(a, b core.Proc) int {
			return cmp.Compare(a.ID, b.ID)
		}
	case "name":
		less = func(a, b core.Proc) int {
			return cmp.Compare(a.Name, b.Name)
		}
	case "status":
		getOrder := func(p core.Proc) int {
			// priority weights
			switch p.Status.Status {
			case core.StatusCreated:
				return 0
			case core.StatusRunning:
				return 100
			case core.StatusStopped:
				return 200
			case core.StatusInvalid:
				return 300
			default:
				return 400
			}
		}
		less = func(a, b core.Proc) int {
			return cmp.Compare(getOrder(a), getOrder(b))
		}
	case "uptime":
		less = cmp2.Comparator[core.Proc](func(a, b core.Proc) int {
			if a.Status.Status != core.StatusRunning || b.Status.Status != core.StatusRunning {
				if a.Status.Status == core.StatusRunning {
					return -1
				}

				if b.Status.Status == core.StatusRunning {
					return 1
				}
			}
			return 0
		}).Then(func(a, b core.Proc) int {
			switch {
			case a.Status.StartTime.Before(b.Status.StartTime):
				return -1
			case b.Status.StartTime.Before(a.Status.StartTime):
				return 1
			default:
				return 0
			}
		})
	default:
		return nil, errors.New("unknown sort field: %q", sortField)
	}

	switch sortOrder {
	case "asc":
		return less, nil
	case "desc":
		return cmp2.Comparator[core.Proc](less).Reversed(), nil
	default:
		return nil, errors.New("unknown sort order: %q", sortOrder)
	}
}

var _usageFlagListFormat = scuf.NewString(func(b scuf.Buffer) {
	b.
		String("Listing format: ").
		Iter(func(yield func(func(scuf.Buffer)) bool) bool {
			for _, format := range _formats {
				yield(func(b scuf.Buffer) {
					b.String(format, scuf.FgYellow).String(", ")
				})
			}
			return false
		}).
		String("any other string is rendred as Go template with ").
		String("core.ProcData", scuf.FgGreen).
		String(" struct")
})

func completeFlagListFormat(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	return fun.FilterMap[string](
		func(format string) (string, bool) {
			return format, strings.HasPrefix(format, prefix)
		},
		_formats...,
	), cobra.ShellCompDirectiveNoFileComp
}

func unmarshalFlagListFormat(format string) (func([]core.Proc) error, error) {
	switch format {
	case _formatTable:
		return func(procsToShow []core.Proc) error {
			renderTable(procsToShow, true)
			return nil
		}, nil
	case _formatCompact:
		return func(procsToShow []core.Proc) error {
			renderTable(procsToShow, false)
			return nil
		}, nil
	case _formatJSON:
		return func(procsToShow []core.Proc) error {
			jsonData, errMarshal := json.MarshalIndent(procsToShow, "", "  ")
			if errMarshal != nil {
				return errors.Wrap(errMarshal, "marshal procs list to json")
			}

			fmt.Println(string(jsonData))
			return nil
		}, nil
	case _formatShort:
		return func(procsToShow []core.Proc) error {
			for _, proc := range procsToShow {
				fmt.Println(proc.Name)
			}
			return nil
		}, nil
	default:
		trimmedFormat := strings.Trim(format, " ")
		finalFormat := strings.
			NewReplacer(
				`\t`, "\t",
				`\n`, "\n",
			).
			Replace(trimmedFormat)

		tmpl, errParse := template.New("list").Parse(finalFormat)
		if errParse != nil {
			return nil, errors.Wrap(errParse, "parse template")
		}

		return func(procsToShow []core.Proc) error {
			var sb strings.Builder
			for _, proc := range procsToShow {
				errRender := tmpl.Execute(&sb, proc)
				if errRender != nil {
					return errors.Wrap(errRender, "format proc line, format=%q: %v", format, proc)
				}

				sb.WriteRune('\n')
			}

			fmt.Println(sb.String())
			return nil
		}, nil
	}
}

var _cmdList = func() *cobra.Command {
	var ids, names, tags []string
	var listFormat, sort string
	cmd := &cobra.Command{
		Use:               "list [name|tag|id]...",
		Short:             "list processes",
		Aliases:           []string{"l", "ls", "ps", "status"},
		GroupID:           "inspection",
		ValidArgsFunction: completeArgGenericSelector,
		RunE: func(cmd *cobra.Command, args []string) error {
			rest := args

			less, err := unmarshalFlagSort(sort)
			if err != nil {
				return fmt.Errorf("unmarshal flag sort: %w", err)
			}

			format, err := unmarshalFlagListFormat(listFormat)
			if err != nil {
				return fmt.Errorf("unmarshal flag format: %w", err)
			}

			app, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrap(errNewApp, "new app")
			}

			procsToShow := app.
				List().
				Filter(core.FilterFunc(
					core.WithAllIfNoFilters,
					core.WithGeneric(rest...),
					core.WithIDs(ids...),
					core.WithNames(names...),
					core.WithTags(tags...),
				)).
				ToSlice()

			slices.SortFunc(procsToShow, less)
			return format(procsToShow)
		},
	}
	cmd.Flags().StringVarP(&listFormat, "format", "f", _formatTable, _usageFlagListFormat)
	cmd.RegisterFlagCompletionFunc("format", completeFlagListFormat)
	cmd.Flags().StringVarP(&sort, "sort", "s", "id:asc", _usageFlagSort)
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	return cmd
}()
