package internal

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aquasecurity/table"
	"github.com/fatih/color"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/db"
)

func init() {
	AllCmds = append(AllCmds, ListCmd)
}

func mapStatus(status db.Status) (string, *int, time.Duration) {
	switch status.Status {
	case db.StatusStarting:
		return color.YellowString("starting"), nil, 0
	case db.StatusRunning:
		return color.GreenString("running"), &status.Pid, time.Since(status.StartTime)
	case db.StatusStopped:
		return color.YellowString("stopped"), nil, 0
	case db.StatusErrored:
		return color.RedString("errored"), nil, 0
	case db.StatusInvalid:
		return color.RedString("invalid(%T)", status), nil, 0
	default:
		return color.RedString("BROKEN(%T)", status), nil, 0
	}
}

var ListCmd = &cli.Command{
	Name:    "list",
	Aliases: []string{"l", "ls", "ps", "status"},
	Usage:   "list processes",
	Flags: []cli.Flag{
		// TODO: list as json
		// TODO: show as tree -t,--tree
		&cli.BoolFlag{ // TODO: ???
			Name:    "mini-list",
			Aliases: []string{"m"},
			Usage:   "display a compacted list without formatting",
		},
		// &cli.BoolFlag{
		// 	Name:  "sort",
		// 	Usage: "sort <id|name|pid>:<inc|dec> sort process according to field value",
		// },
		&cli.BoolFlag{
			Name:  "compact",
			Usage: "show compact table",
			Value: false,
		},
		// &cli.StringFlag{
		// 	Name:    "format",
		// 	Aliases: []string{"f"},
		// 	Usage:   "Go template string to use for formatting",
		// },
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: "name(s) of process(es) to stop",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "tag(s) of process(es) to stop",
		},
		&cli.Uint64SliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to stop",
		},
		&cli.StringSliceFlag{
			Name:  "status",
			Usage: "status(es) of process(es) to stop",
		},
	},
	Action: func(ctx *cli.Context) error {
		return list(
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.StringSlice("status"),
			ctx.Uint64Slice("id"),
			ctx.Bool("compact"),
		)
	},
}

func list(
	genericFilters, nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
	compact bool,
) error {
	resp, err := db.New(_daemonDBFile).List()
	if err != nil {
		return err
	}

	procIDsToShow := internal.FilterProcs[db.ProcID](
		resp,
		internal.WithAllIfNoFilters,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithStatuses(statusFilters),
		internal.WithTags(tagFilters),
	)

	procsToShow := internal.MapDict(procIDsToShow, resp)
	sort.Slice(procsToShow, func(i, j int) bool {
		return procsToShow[i].ID < procsToShow[j].ID
	})

	t := table.New(os.Stdout)
	t.SetRowLines(!compact)
	t.SetDividers(table.UnicodeRoundedDividers)
	t.SetHeaders("id", "name", "status", "pid", "uptime", "tags", "cpu", "memory", "cmd")
	t.SetHeaderStyle(table.StyleBold)
	t.SetLineStyle(table.StyleDim)

	// TODO: sort
	for _, proc := range procsToShow {
		status, pid, uptime := mapStatus(proc.Status)
		t.AddRow(
			color.New(color.FgCyan, color.Bold).Sprint(proc.ID),
			proc.Name,
			status,
			internal.IfNotNil(pid, strconv.Itoa),
			lo.If(pid == nil, "").
				Else(uptime.Truncate(time.Second).String()),
			fmt.Sprint(proc.Tags),
			fmt.Sprint(proc.Status.Cpu),
			fmt.Sprint(proc.Status.Memory),
			fmt.Sprintf("%s %s", proc.Command, strings.Join(proc.Args, " ")), // TODO: escape args
		)
	}

	t.Render()

	return nil
}
