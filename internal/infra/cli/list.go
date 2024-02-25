package cli

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/aquasecurity/table"
	"github.com/kballard/go-shellquote"
	"github.com/rprtr258/cli"
	cmp2 "github.com/rprtr258/cmp"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/scuf"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

const (
	_formatTable   = "table"
	_formatCompact = "compact"
	_formatJSON    = "json"
	_formatShort   = "short"
)

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

type flagListSort struct {
	less func(a, b core.Proc) int
}

func (f *flagListSort) Usage() string {
	return scuf.NewString(func(b scuf.Buffer) {
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
}

func (f *flagListSort) UnmarshalFlag(value string) error {
	sortField := value
	sortOrder := "asc"
	if i := strings.IndexRune(sortField, ':'); i != -1 {
		sortField, sortOrder = sortField[:i], sortField[i+1:]
	}

	switch sortField {
	case "id":
		f.less = func(a, b core.Proc) int {
			return cmp.Compare(a.ID, b.ID)
		}
	case "name":
		f.less = func(a, b core.Proc) int {
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
		f.less = func(a, b core.Proc) int {
			return cmp.Compare(getOrder(a), getOrder(b))
		}
	case "uptime":
		f.less = cmp2.Comparator[core.Proc](func(a, b core.Proc) int {
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
		return errors.New("unknown sort field: %q", sortField)
	}

	switch sortOrder {
	case "asc":
	case "desc":
		oldSortFunc := f.less
		f.less = cmp2.Comparator[core.Proc](oldSortFunc).Reversed()
	default:
		return errors.New("unknown sort order: %q", sortOrder)
	}

	return nil
}

type flagListFormat struct {
	f func([]core.Proc) error
}

func (*flagListFormat) Usage() string {
	return scuf.NewString(func(b scuf.Buffer) {
		b.
			String("Listing format: ").
			String(_formatTable, scuf.FgYellow).String(", ").
			String(_formatCompact, scuf.FgYellow).String(", ").
			String(_formatJSON, scuf.FgYellow).String(", ").
			String(_formatShort, scuf.FgYellow).
			String(", any other string is rendred as Go template with ").
			String("core.ProcData", scuf.FgGreen).
			String(" struct")
	})
}

func (f *flagListFormat) Complete(prefix string) []cli.Completion {
	return fun.FilterMap[cli.Completion](
		func(format string) (cli.Completion, bool) {
			return cli.Completion{
				Item:        format,
				Description: fun.Invalid[string](),
			}, strings.HasPrefix(format, prefix)
		},
		_formatTable,
		_formatCompact,
		_formatJSON,
		_formatShort,
	)
}

func (f *flagListFormat) UnmarshalFlag(format string) error {
	switch format {
	case _formatTable:
		f.f = func(procsToShow []core.Proc) error {
			renderTable(procsToShow, true)
			return nil
		}
	case _formatCompact:
		f.f = func(procsToShow []core.Proc) error {
			renderTable(procsToShow, false)
			return nil
		}
	case _formatJSON:
		f.f = func(procsToShow []core.Proc) error {
			jsonData, errMarshal := json.MarshalIndent(procsToShow, "", "  ")
			if errMarshal != nil {
				return errors.Wrap(errMarshal, "marshal procs list to json")
			}

			fmt.Println(string(jsonData))
			return nil
		}
	case _formatShort:
		f.f = func(procsToShow []core.Proc) error {
			for _, proc := range procsToShow {
				fmt.Println(proc.Name)
			}
			return nil
		}
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
			return errors.Wrap(errParse, "parse template")
		}

		f.f = func(procsToShow []core.Proc) error {
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
		}
	}
	return nil
}

type _cmdList struct {
	// cmd.FindOptionByLongName("format").Description = (*flagListFormat)(nil).Usage() // TODO: use as flag type method
	Format flagListFormat `short:"f" long:"format" default:"table"`
	// cmd.FindOptionByLongName("sort").Description = (*flagListSort)(nil).Usage()     // TODO: use as flag type method
	Sort  flagListSort   `short:"s" long:"sort" default:"id:asc"`
	Names []flagProcName `long:"name" description:"name(s) of process(es) to list"`
	Tags  []flagProcTag  `long:"tag" description:"tag(s) of process(es) to list"`
	IDs   []flagPMID     `long:"id" description:"id(s) of process(es) to list"`
	Args  struct {
		Rest []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
}

func (x _cmdList) Execute(ctx context.Context) error {
	app, errNewApp := app.New()
	if errNewApp != nil {
		return errors.Wrap(errNewApp, "new app")
	}

	procsToShow := app.
		List().
		Filter(core.FilterFunc(
			core.WithAllIfNoFilters,
			core.WithGeneric(x.Args.Rest...),
			core.WithIDs(x.IDs...),
			core.WithNames(x.Names...),
			core.WithTags(x.Tags...),
		)).
		ToSlice()

	slices.SortFunc(procsToShow, func(i, j core.Proc) int {
		return x.Sort.less(i, j)
	})

	return x.Format.f(procsToShow)
}
