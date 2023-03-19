package internal

import (
	"context"
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

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
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
		// &cli.StringFlag{
		// 	Name:    "format",
		// 	Aliases: []string{"f"},
		// 	Usage:   "Go template string to use for formatting",
		// },
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
		&cli.StringSliceFlag{
			Name:  "status",
			Usage: "status(es) of process(es) to list",
		},
	},
	Action: func(ctx *cli.Context) error {
		return list(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.StringSlice("status"),
			ctx.Uint64Slice("id"),
		)
	},
}

func list(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
) error {
	client, err := client.NewGrpcClient()
	if err != nil {
		return xerr.NewWM(err, "new grpc client")
	}
	defer deferErr(client.Close)()

	resp, err := client.List(ctx)
	if err != nil {
		return xerr.NewWM(err, "list server call")
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
		return procsToShow[i].ProcID < procsToShow[j].ProcID
	})

	procsTable := table.New(os.Stdout)
	procsTable.SetDividers(table.UnicodeRoundedDividers)
	procsTable.SetHeaders("id", "name", "status", "pid", "uptime", "tags", "cpu", "memory", "cmd")
	procsTable.SetHeaderStyle(table.StyleBold)
	procsTable.SetLineStyle(table.StyleDim)
	for _, proc := range procsToShow {
		status, pid, uptime := mapStatus(proc.Status)
		procsTable.AddRow(
			color.New(color.FgCyan, color.Bold).Sprint(proc.ProcID),
			proc.Name,
			status,
			internal.IfNotNil(pid, strconv.Itoa),
			lo.If(pid == nil, "").
				Else(uptime.Truncate(time.Second).String()),
			fmt.Sprint(proc.Tags),
			fmt.Sprint(proc.Status.CPU),
			fmt.Sprint(proc.Status.Memory),
			fmt.Sprintf("%s %s", proc.Command, strings.Join(proc.Args, " ")), // TODO: escape args
		)
	}
	procsTable.Render()

	return nil
}
