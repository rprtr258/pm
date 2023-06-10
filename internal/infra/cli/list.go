package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/aquasecurity/table"
	"github.com/fatih/color"
	"github.com/kballard/go-shellquote"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/fun"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/pkg/client"
)

func mapStatus(status db.Status) (string, *int, time.Duration) {
	switch status.Status {
	case db.StatusStarting:
		return color.YellowString("starting"), nil, 0
	case db.StatusRunning:
		return color.GreenString("running"), &status.Pid, time.Since(status.StartTime)
	case db.StatusStopped:
		return color.YellowString("stopped(%d)", status.ExitCode), nil, 0
	case db.StatusInvalid:
		return color.RedString("invalid(%T)", status), nil, 0
	default:
		return color.RedString("BROKEN(%T)", status), nil, 0
	}
}

var _listCmd = &cli.Command{
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

	procIDsToShow := core.FilterProcs[db.ProcID](
		resp,
		core.WithAllIfNoFilters,
		core.WithGeneric(genericFilters),
		core.WithIDs(idFilters),
		core.WithNames(nameFilters),
		core.WithStatuses(statusFilters),
		core.WithTags(tagFilters),
	)

	procsToShow := fun.MapDict(procIDsToShow, resp)
	sort.Slice(procsToShow, func(i, j int) bool {
		return procsToShow[i].ProcID < procsToShow[j].ProcID
	})

	procsTable := table.New(os.Stdout)
	procsTable.SetDividers(table.UnicodeRoundedDividers)
	procsTable.SetHeaders("id", "name", "status", "pid", "uptime", "tags", "cpu", "memory", "cmd")
	procsTable.SetHeaderStyle(table.StyleBold)
	procsTable.SetLineStyle(table.StyleDim)
	for _, proc := range procsToShow {
		// TODO: if errored/stopped show time since start instead of uptime (not in place of)
		status, pid, uptime := mapStatus(proc.Status)

		procsTable.AddRow(
			color.New(color.FgCyan, color.Bold).Sprint(proc.ProcID),
			proc.Name,
			status,
			strconv.Itoa(fun.Deref(pid)),
			// TODO: check status instead for following parameters
			fun.If(pid == nil, "").Else(uptime.Truncate(time.Second).String()),
			fmt.Sprint(proc.Tags),
			fmt.Sprint(proc.Status.CPU),
			fmt.Sprint(proc.Status.Memory),
			shellquote.Join(append([]string{proc.Command}, proc.Args...)...),
		)
	}
	procsTable.Render()

	return nil
}
