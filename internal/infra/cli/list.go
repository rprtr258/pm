package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/aquasecurity/table"
	"github.com/kballard/go-shellquote"
	flags "github.com/rprtr258/cli/contrib"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/rprtr258/xerr"

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
	less func(a, b core.Proc) bool
}

func (f *flagListSort) Usage() string {
	return scuf.NewString(func(b scuf.Buffer) {
		b.
			String("Sort order. Available sort fields: ").
			String("id", scuf.FgYellow).String(", ").
			String("name", scuf.FgYellow).String(", ").
			String("status", scuf.FgYellow).String(", ").
			String("pid", scuf.FgYellow).String(", ").
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
		f.less = func(a, b core.Proc) bool {
			return a.ID < b.ID
		}
	case "name":
		f.less = func(a, b core.Proc) bool {
			return a.Name < b.Name
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
		f.less = func(a, b core.Proc) bool {
			return getOrder(a) < getOrder(b)
		}
	case "pid":
		f.less = func(a, b core.Proc) bool {
			if a.Status.Status != core.StatusRunning || b.Status.Status != core.StatusRunning {
				if a.Status.Status == core.StatusRunning {
					return true
				}

				if b.Status.Status == core.StatusRunning {
					return false
				}

				return a.ID < b.ID
			}

			return a.ID < b.ID
		}
	case "uptime":
		now := time.Now()
		f.less = func(a, b core.Proc) bool {
			if a.Status.Status != core.StatusRunning || b.Status.Status != core.StatusRunning {
				if a.Status.Status == core.StatusRunning {
					return true
				}

				if b.Status.Status == core.StatusRunning {
					return false
				}

				return a.ID < b.ID
			}

			return a.Status.StartTime.Sub(now) < b.Status.StartTime.Sub(now)
		}
	default:
		return xerr.NewM("unknown sort field", xerr.Fields{"field": sortField})
	}

	switch sortOrder {
	case "asc":
	case "desc":
		oldSortFunc := f.less
		f.less = func(a, b core.Proc) bool {
			return !oldSortFunc(a, b)
		}
	default:
		return xerr.NewM("unknown sort order", xerr.Fields{"order": sortOrder})
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

func (f *flagListFormat) Complete(prefix string) []flags.Completion {
	return fun.FilterMap[flags.Completion](
		func(format string) (flags.Completion, bool) {
			return flags.Completion{
				Item:        format,
				Description: "",
			}, strings.HasPrefix(format, prefix)
		},
		_formatTable,
		_formatCompact,
		_formatJSON,
		_formatShort,
	)
}

func (f *flagListFormat) UnmarshalFlag(value string) error {
	switch value {
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
				return xerr.NewWM(errMarshal, "marshal procs list to json")
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
		trimmedFormat := strings.Trim(value, " ")
		finalFormat := strings.
			NewReplacer(
				`\t`, "\t",
				`\n`, "\n",
			).
			Replace(trimmedFormat)

		tmpl, errParse := template.New("list").Parse(finalFormat)
		if errParse != nil {
			return xerr.NewWM(errParse, "parse template")
		}

		f.f = func(procsToShow []core.Proc) error {
			var sb strings.Builder
			for _, proc := range procsToShow {
				errRender := tmpl.Execute(&sb, proc)
				if errRender != nil {
					return xerr.NewWM(errRender, "format proc line", xerr.Fields{
						"format": value,
						"proc":   proc,
					})
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
	Format flagListFormat `short:"f" long:"format" default:"table"`
	Sort   flagListSort   `short:"s" long:"sort" default:"id:asc"`
	Names  []flagProcName `long:"name" description:"name(s) of process(es) to list"`
	Tags   []flagProcTag  `long:"tag" description:"tag(s) of process(es) to list"`
	IDs    []flagPMID     `long:"id" description:"id(s) of process(es) to list"`
	Args   struct {
		Rest []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
}

func (x *_cmdList) Execute(_ []string) error {
	app, errNewApp := app.New()
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	list := app.List() // TODO: move in filters which are bit below

	procIDsToShow := core.FilterProcMap(
		list,
		core.WithAllIfNoFilters,
		core.WithGeneric(x.Args.Rest...),
		core.WithIDs(x.IDs...),
		core.WithNames(x.Names...),
		core.WithTags(x.Tags...),
	)

	procsToShow := fun.MapDict(list, procIDsToShow...)
	sort.Slice(procsToShow, func(i, j int) bool {
		return x.Sort.less(procsToShow[i], procsToShow[j])
	})

	return x.Format.f(procsToShow)
}
