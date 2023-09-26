package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aquasecurity/table"
	"github.com/kballard/go-shellquote"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/internal/infra/cli/log/buffer"
	"github.com/rprtr258/pm/internal/infra/daemon"
	"github.com/rprtr258/pm/pkg/client"
)

const (
	_formatTable   = "table"
	_formatCompact = "compact"
	_formatJSON    = "json"
	_formatShort   = "short"
)

var _listCmd = &cli.Command{
	Name:     "list",
	Aliases:  []string{"l", "ls", "ps", "status"},
	Usage:    "list processes",
	Category: "inspection",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "format",
			Aliases: []string{"f"},
			Usage: buffer.NewString(func(b *buffer.Buffer) {
				b.
					String("Listing format: ").
					String(_formatTable, buffer.FgYellow).String(", ").
					String(_formatCompact, buffer.FgYellow).String(", ").
					String(_formatJSON, buffer.FgYellow).String(", ").
					String(_formatShort, buffer.FgYellow).
					String(", any other string is rendred as Go template with ").
					String("core.ProcData", buffer.FgGreen).
					String(" struct")
			}),
			Value: "table",
		},
		&cli.StringFlag{
			Name:    "sort",
			Aliases: []string{"s"},
			Usage: buffer.NewString(func(b *buffer.Buffer) {
				b.
					String("Sort order. Available sort fields: ").
					String("id", buffer.FgYellow).String(", ").
					String("name", buffer.FgYellow).String(", ").
					String("status", buffer.FgYellow).String(", ").
					String("pid", buffer.FgYellow).String(", ").
					String("uptime", buffer.FgYellow).
					String(". Order can be changed by adding ").
					String(":asc", buffer.FgRed).
					String(" or ").
					String(":desc", buffer.FgRed)
			}),
			Value: "id:asc",
		},
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: "name(s) of process(es) to list",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "tag(s) of process(es) to list",
		},
		&cli.Uint64SliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to list",
		},
	},
	Action: func(ctx *cli.Context) error {
		if errDaemon := daemon.EnsureRunning(ctx.Context); errDaemon != nil {
			return xerr.NewWM(errDaemon, "ensure daemon is running")
		}

		sortField := ctx.String("sort")
		sortOrder := "asc"
		if i := strings.IndexRune(sortField, ':'); i != -1 {
			sortField, sortOrder = sortField[:i], sortField[i+1:]
		}

		var sortFunc func(a, b core.Proc) bool
		switch sortField {
		case "id":
			sortFunc = func(a, b core.Proc) bool {
				return a.ID < b.ID
			}
		case "name":
			sortFunc = func(a, b core.Proc) bool {
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
			sortFunc = func(a, b core.Proc) bool {
				return getOrder(a) < getOrder(b)
			}
		case "pid":
			sortFunc = func(a, b core.Proc) bool {
				if a.Status.Status != core.StatusRunning || b.Status.Status != core.StatusRunning {
					if a.Status.Status == core.StatusRunning {
						return true
					}

					if b.Status.Status == core.StatusRunning {
						return false
					}

					return a.ID < b.ID
				}

				return a.Status.Pid < b.Status.Pid
			}
		case "uptime":
			now := time.Now()
			sortFunc = func(a, b core.Proc) bool {
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
			oldSortFunc := sortFunc
			sortFunc = func(a, b core.Proc) bool {
				return !oldSortFunc(a, b)
			}
		default:
			return xerr.NewM("unknown sort order", xerr.Fields{"order": sortOrder})
		}

		return list(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.Uint64Slice("id"),
			ctx.String("format"),
			sortFunc,
		)
	},
}

func mapStatus(status core.Status) (string, *int, time.Duration) {
	switch status.Status {
	case core.StatusCreated:
		return buffer.String("created", buffer.FgYellow), nil, 0
	case core.StatusRunning:
		return buffer.String("running", buffer.FgGreen), &status.Pid, time.Since(status.StartTime)
	case core.StatusStopped:
		return buffer.String("stopped", buffer.FgRed), nil, 0
	case core.StatusInvalid:
		return buffer.String(fmt.Sprintf("invalid(%#v)", status.Status), buffer.FgRed), nil, 0
	default:
		return buffer.String(fmt.Sprintf("BROKEN(%T)", status), buffer.FgRed), nil, 0
	}
}

func renderTable(procs []core.Proc, setRowLines bool) {
	procsTable := table.New(os.Stdout)
	procsTable.SetDividers(table.UnicodeRoundedDividers)
	procsTable.SetHeaders("id", "name", "status", "pid", "uptime", "tags" /*"cpu", "memory",*/, "cmd")
	procsTable.SetHeaderStyle(table.StyleBold)
	procsTable.SetLineStyle(table.StyleDim)
	procsTable.SetRowLines(setRowLines)
	for _, proc := range procs {
		// TODO: if errored/stopped show time since start instead of uptime (not in place of)
		status, pid, uptime := mapStatus(proc.Status)

		procsTable.AddRow(
			buffer.String(fmt.Sprint(proc.ID), buffer.FgCyan, buffer.ColorBold),
			proc.Name,
			status,
			fun.
				If(pid == nil, "").
				ElseF(func() string {
					return strconv.Itoa(*pid)
				}),
			// TODO: check status instead for following parameters
			fun.If(pid == nil, "").Else(uptime.Truncate(time.Second).String()),
			strings.Join(proc.Tags, "\n"),
			// fmt.Sprint(proc.Status.CPU),
			// fmt.Sprint(proc.Status.Memory),
			shellquote.Join(append([]string{proc.Command}, proc.Args...)...),
		)
	}
	procsTable.Render()
}

func list(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []core.ProcID,
	format string,
	sortFunc func(a, b core.Proc) bool,
) error {
	pmClient, err := client.New()
	if err != nil {
		return xerr.NewWM(err, "new grpc client")
	}
	defer deferErr(pmClient.Close)()

	app, errNewApp := pm.New(pmClient)
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	list, err := app.List(ctx) // TODO: move in filters which are bit below
	if err != nil {
		return xerr.NewWM(err, "list server call")
	}

	procIDsToShow := core.FilterProcMap(
		list,
		core.NewFilter(
			core.WithAllIfNoFilters,
			core.WithGeneric(genericFilters),
			core.WithIDs(idFilters...),
			core.WithNames(nameFilters),
			core.WithTags(tagFilters),
		),
	)

	procsToShow := fun.MapDict(procIDsToShow, list)
	sort.Slice(procsToShow, func(i, j int) bool {
		return sortFunc(procsToShow[i], procsToShow[j])
	})

	switch format {
	case _formatTable:
		renderTable(procsToShow, true)
	case _formatCompact:
		renderTable(procsToShow, false)
	case _formatJSON:
		jsonData, errMarshal := json.MarshalIndent(procsToShow, "", "  ")
		if errMarshal != nil {
			return xerr.NewWM(errMarshal, "marshal procs list to json")
		}

		fmt.Println(string(jsonData))
	case _formatShort:
		for _, proc := range procsToShow {
			fmt.Println(proc.Name)
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
			return xerr.NewWM(errParse, "parse template")
		}

		var sb strings.Builder
		for _, proc := range procsToShow {
			errRender := tmpl.Execute(&sb, proc)
			if errRender != nil {
				return xerr.NewWM(errRender, "format proc line", xerr.Fields{
					"format": format,
					"proc":   proc,
				})
			}

			sb.WriteRune('\n')
		}

		fmt.Println(sb.String())
	}

	return nil
}
