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

	cmp2 "github.com/rprtr258/cmp"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
	"github.com/rprtr258/pm/internal/table"
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

func mapStatus(status core.Status) string {
	var color scuf.Modifier
	switch status {
	case core.StatusCreated:
		color = scuf.FgHiYellow
	case core.StatusRunning:
		color = scuf.FgHiGreen
	case core.StatusStopped:
		color = scuf.Combine(scuf.FgRed, scuf.ModBold)
	}
	return scuf.String(status.String(), color)
}

func formatMemory(m uint64) string {
	switch {
	case m < 1024:
		return fmt.Sprintf("%.2f B", float64(m))
	case m < 1024*1024:
		return fmt.Sprintf("%.2f KB", float64(m)/1024)
	case m < 1024*1024*1024:
		return fmt.Sprintf("%.2f MB", float64(m)/1024/1024)
	default:
		return fmt.Sprintf("%.2f GB", float64(m)/1024/1024/1024)
	}
}

func commonPrefixLength(s, t core.PMID) int {
	res := 0
	for i := 0; i < len(s) && i < len(t); i++ {
		if s[i] != t[i] {
			break
		}
		res++
	}
	return res
}

func shortIDs(procs []core.ProcStat) []string {
	if len(procs) == 0 {
		return nil
	}
	if len(procs) == 1 {
		return []string{string(procs[0].ID.String()[0])}
	}

	idx := fun.SliceToMap[core.PMID, int](func(proc core.ProcStat, i int) (core.PMID, int) {
		return proc.ID, i
	}, procs...)

	ids := fun.Map[core.PMID](func(proc core.ProcStat) core.PMID { return proc.ID }, procs...)
	slices.Sort(ids)

	res := make([]string, len(procs))
	for i, id := range ids {
		var prefix int
		if i < len(ids)-1 {
			prefix = commonPrefixLength(id, ids[i+1]) + 1
		}
		if i > 0 {
			prefix = max(prefix, commonPrefixLength(id, ids[i-1])+1)
		}
		res[idx[id]] = id.String()[:prefix]
	}
	return res
}

func renderTable(procs []core.ProcStat, showRowDividers bool) {
	ids := shortIDs(procs)
	t := table.Table{
		Headers: fun.Map[string](func(col string) string {
			return scuf.String(col, scuf.ModBold)
		}, "id", "name", "status", "uptime", "tags", "cpu", "memory"),
		Rows: fun.Map[[]string](func(proc core.ProcStat, i int) []string {
			// TODO: if errored/stopped show time since start instead of uptime (not in place of)
			uptime := time.Duration(0)
			if stat, ok := linuxprocess.StatPMID(proc.ID, app.EnvPMID); ok {
				uptime = time.Since(stat.ChildStartTime)
			}

			return []string{
				scuf.String(ids[i], scuf.FgCyan, scuf.ModBold),
				proc.Name,
				mapStatus(proc.Status),
				fun.
					If(proc.Status != core.StatusRunning, "").
					Else(uptime.Truncate(time.Second).String()),
				strings.Join(proc.Tags, " "),
				fmt.Sprint(proc.CPU) + "%",
				formatMemory(proc.Memory),
			}
		}, procs...),
		HaveInnerRowsDividers: showRowDividers,
	}

	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	fmt.Println(table.Render(t, width))
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

func unmarshalFlagSort(value string) (func(a, b core.ProcStat) int, error) {
	sortField := value
	sortOrder := "asc"
	if i := strings.IndexRune(sortField, ':'); i != -1 {
		sortField, sortOrder = sortField[:i], sortField[i+1:]
	}

	var less func(a, b core.ProcStat) int
	switch sortField {
	case "id":
		less = func(a, b core.ProcStat) int {
			return cmp.Compare(a.ID, b.ID)
		}
	case "name":
		less = func(a, b core.ProcStat) int {
			return cmp.Compare(a.Name, b.Name)
		}
	case "status":
		getOrder := func(p core.ProcStat) int {
			// priority weights
			switch p.Status {
			case core.StatusCreated:
				return 0
			case core.StatusRunning:
				return 100
			case core.StatusStopped:
				return 200
			default:
				return 400
			}
		}
		less = func(a, b core.ProcStat) int {
			return cmp.Compare(getOrder(a), getOrder(b))
		}
	case "uptime":
		less = cmp2.Comparator[core.ProcStat](func(a, b core.ProcStat) int {
			if a.Status != core.StatusRunning || b.Status != core.StatusRunning {
				if a.Status == core.StatusRunning {
					return -1
				}

				if b.Status == core.StatusRunning {
					return 1
				}
			}
			return 0
		}).Then(func(a, b core.ProcStat) int {
			switch {
			case a.StartTime.Before(b.StartTime):
				return -1
			case b.StartTime.Before(a.StartTime):
				return 1
			default:
				return 0
			}
		})
	default:
		return nil, errors.Newf("unknown sort field: %q", sortField)
	}

	switch sortOrder {
	case "asc":
		return less, nil
	case "desc":
		return cmp2.Comparator[core.ProcStat](less).Reversed(), nil
	default:
		return nil, errors.Newf("unknown sort order: %q", sortOrder)
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
		String("any other string is rendered as Go template with ").
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

func unmarshalFlagListFormat(format string) (func([]core.ProcStat) error, error) {
	switch format {
	case _formatTable:
		return func(procsToShow []core.ProcStat) error {
			renderTable(procsToShow, true)
			return nil
		}, nil
	case _formatCompact:
		return func(procsToShow []core.ProcStat) error {
			renderTable(procsToShow, false)
			return nil
		}, nil
	case _formatJSON:
		return func(procsToShow []core.ProcStat) error {
			jsonData, errMarshal := json.MarshalIndent(procsToShow, "", "  ")
			if errMarshal != nil {
				return errors.Wrapf(errMarshal, "marshal procs list to json")
			}

			fmt.Println(string(jsonData))
			return nil
		}, nil
	case _formatShort:
		return func(procsToShow []core.ProcStat) error {
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
			return nil, errors.Wrapf(errParse, "parse template")
		}

		return func(procsToShow []core.ProcStat) error {
			var sb strings.Builder
			for _, proc := range procsToShow {
				errRender := tmpl.Execute(&sb, proc)
				if errRender != nil {
					return errors.Wrapf(errRender, "format proc line, format=%q: %v", format, proc)
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
		RunE: func(_ *cobra.Command, args []string) error {
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
				return errors.Wrapf(errNewApp, "new app")
			}

			filterFunc := core.FilterFunc(
				core.WithAllIfNoFilters,
				core.WithGeneric(args...),
				core.WithIDs(ids...),
				core.WithNames(names...),
				core.WithTags(tags...),
			)
			procsToShow := app.
				List().
				Filter(func(ps core.ProcStat) bool {
					return filterFunc(ps.Proc)
				}).
				ToSlice()

			if len(procsToShow) == 0 {
				fmt.Fprintln(os.Stderr, "no processes added")
				return nil
			}

			slices.SortFunc(procsToShow, less)
			return format(procsToShow)
		},
	}
	cmd.Flags().StringVarP(&listFormat, "format", "f", _formatTable, _usageFlagListFormat)
	registerFlagCompletionFunc(cmd, "format", completeFlagListFormat)
	cmd.Flags().StringVarP(&sort, "sort", "s", "id:asc", _usageFlagSort)
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	return cmd
}()
