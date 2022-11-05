package internal

import (
	"fmt"
	"os"
	"strconv"
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

func mapStatus(status db.Status) string {
	switch status.Status {
	case db.StatusStarting:
		return color.YellowString("starting")
	case db.StatusRunning:
		// TODO: separate
		return color.GreenString(
			"running(pid=%d,uptime=%v)",
			status.Pid,
			time.Since(status.StartTime),
		)
	case db.StatusStopped:
		return color.YellowString("stopped")
	case db.StatusErrored:
		return color.RedString("errored")
	case db.StatusInvalid:
		return color.RedString("invalid(%T)", status)
	default:
		return color.RedString("BROKEN(%T)", status)
	}
}

var ListCmd = &cli.Command{
	Name:    "list",
	Aliases: []string{"l"},
	Flags: []cli.Flag{
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
		// t.SetAutoMerge(true)
		t.SetHeaders("id", "name", "status", "tags", "cpu", "memory", "cmd")
		t.SetHeaderStyle(table.StyleBold)
		t.SetLineStyle(table.StyleDim)

		for _, item := range procsToShow {
			t.AddRow(
				color.New(color.FgCyan, color.Bold).Sprint(item.ID),
				item.Name,
				mapStatus(item.Status),
				fmt.Sprint(item.Tags),
				fmt.Sprint(item.Status.Cpu),
				fmt.Sprint(item.Status.Memory),
				item.Cmd,
			)
		}

		t.Render()

		return nil
	},
}

// TODO: any other filters?
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
