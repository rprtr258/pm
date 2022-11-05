package internal

import (
	"fmt"
	"os"
	"time"

	"github.com/aquasecurity/table"
	"github.com/fatih/color"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/daemon"
)

func init() {
	AllCmds = append(AllCmds, ListCmd)
}

func mapStatus(status daemon.Status) string {
	switch status.Status {
	case daemon.StatusStarting:
		return color.YellowString("starting")
	case daemon.StatusRunning:
		// TODO: separate
		return color.GreenString(
			"running(pid=%d,uptime=%v)",
			status.Pid,
			time.Since(status.StartTime),
		)
	case daemon.StatusStopped:
		return color.YellowString("stopped")
	case daemon.StatusErrored:
		return color.RedString("errored")
	case daemon.StatusInvalid:
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
		db, err := daemon.New(_daemonDBFile)
		if err != nil {
			return err
		}
		defer db.Close()

		resp, err := db.List()
		if err != nil {
			return err
		}

		t := table.New(os.Stdout)
		t.SetRowLines(!ctx.Bool("compact"))
		t.SetDividers(table.UnicodeRoundedDividers)
		t.SetAutoMerge(true)
		t.SetHeaders("id", "name", "status", "tags", "cpu", "memory", "cmd")
		t.SetHeaderStyle(table.StyleBold)
		t.SetLineStyle(table.StyleDim)

		lo.ForEach(resp, func(item daemon.ProcData, _ int) {
			t.AddRow(
				color.New(color.FgCyan, color.Bold).Sprint(item.ID),
				item.Name,
				mapStatus(item.Status),
				fmt.Sprint(item.Tags),
				fmt.Sprint(item.Status.Cpu),
				fmt.Sprint(item.Status.Memory),
				item.Cmd,
			)
		})

		t.Render()

		return nil
	},
}
