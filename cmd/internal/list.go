package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aquasecurity/table"
	"github.com/fatih/color"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/db"
)

func init() {
	AllCmds = append(AllCmds, ListCmd)
}

func mapStatus(status db.Status) (string, *uint64, time.Duration) {
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
		&cli.BoolFlag{
			Name:  "sort",
			Usage: "sort <id|name|pid>:<inc|dec> sort process according to field value",
		},
		&cli.BoolFlag{
			Name:  "compact",
			Usage: "show compact table",
			Value: false,
		},
		&cli.StringFlag{
			Name:    "format",
			Aliases: []string{"f"},
			Usage:   "Go template string to use for formatting",
		},
	},
	Action: func(ctx *cli.Context) error {
		resp, err := db.New(_daemonDBFile).List()
		if err != nil {
			return err
		}

		procsToShow := filterProcs(
			resp,
			ctx.Args().Slice(),
			[]string{},
			[]string{},
			[]db.ProcStatus{},
			[]db.ProcID{},
		)

		t := table.New(os.Stdout)
		t.SetRowLines(!ctx.Bool("compact"))
		t.SetDividers(table.UnicodeRoundedDividers)
		t.SetHeaders("id", "name", "status", "pid", "uptime", "tags", "cpu", "memory", "cmd")
		t.SetHeaderStyle(table.StyleBold)
		t.SetLineStyle(table.StyleDim)

		for _, item := range procsToShow {
			status, pid, uptime := mapStatus(item.Status)
			t.AddRow(
				color.New(color.FgCyan, color.Bold).Sprint(item.ID),
				item.Name,
				status,
				lo.If(pid == nil, "").
					ElseF(func() string { return strconv.FormatUint(*pid, 10) }),
				lo.If(pid == nil, "").
					Else(uptime.String()),
				fmt.Sprint(item.Tags),
				fmt.Sprint(item.Status.Cpu),
				fmt.Sprint(item.Status.Memory),
				fmt.Sprintf("%s %s", item.Command, strings.Join(item.Args, " ")), // TODO: escape args
			)
		}

		t.Render()

		return nil
	},
}

func filterProcs(
	procs []db.ProcData,
	generic, names, tags []string,
	statuses []db.ProcStatus,
	ids []db.ProcID,
) []db.ProcData {
	// if no filters, return all
	if len(generic) == 0 &&
		len(names) == 0 &&
		len(tags) == 0 &&
		len(statuses) == 0 &&
		len(ids) == 0 {
		return procs
	}

	genericIDs := lo.FilterMap(generic, func(filter string, _ int) (db.ProcID, bool) {
		id, err := strconv.ParseUint(filter, 10, 64)
		if err != nil {
			return 0, false
		}

		return db.ProcID(id), true
	})

	return lo.Filter(procs, func(proc db.ProcData, _ int) bool {
		return lo.Contains(names, proc.Name) ||
			lo.Some(tags, proc.Tags) ||
			lo.Contains(statuses, proc.Status.Status) ||
			lo.Contains(ids, proc.ID) ||
			lo.Contains(generic, proc.Name) ||
			lo.Some(generic, proc.Tags) ||
			lo.Contains(genericIDs, proc.ID)
	})
}
