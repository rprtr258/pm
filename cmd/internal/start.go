package internal

import (
	"context"
	"fmt"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/db"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
)

func init() {
	AllCmds = append(AllCmds, StartCmd)
}

var StartCmd = &cli.Command{
	Name:      "start",
	ArgsUsage: "<name|tag|id|status>...",
	Usage:     "start process and manage it",
	Flags: []cli.Flag{
		// &cli.BoolFlag{Name:        "only", Usage: "with json declaration, allow to only act on one application"},
		// &cli.BoolFlag{Name:        "watch", Usage: "Watch folder for changes"},
		// &cli.StringSliceFlag{Name: "watch", Usage: "watch application folder for changes"},
		// &cli.StringSliceFlag{Name: "ext", Usage: "watch only this file extensions"},
		// &cli.StringFlag{Name:      "cron-restart", Aliases: []string{"c", "cron"}, Usage: "restart a running process based on a cron pattern"},
		// &cli.BoolFlag{Name:        "wait-ready", Usage: "ask pm2 to wait for ready event from your app"},
		// &cli.DurationFlag{Name:    "watch-delay", Usage: "specify a restart delay after changing files (--watch-delay 4 (in sec) or 4000ms)"},
		// &cli.BoolFlag{Name:        "output", Aliases: []string{"o"}, Usage: "specify log file for stdout"},
		// &cli.PathFlag{Name:        "error", Aliases: []string{"e"}, Usage: "specify log file for stderr"},
		// &cli.PathFlag{Name:        "log", Aliases: []string{"l"}, Usage: "specify log file which gathers both stdout and stderr"},
		// &cli.BoolFlag{Name:        "disable-logs", Usage: "disable all logs storage"},
		// &cli.StringFlag{Name:      "log-date-format", Usage: "add custom prefix timestamp to logs"},
		// &cli.BoolFlag{Name:        "time", Usage: "enable time logging"},
		// &cli.StringFlag{Name:      "log-type", Usage: "specify log output style (raw by default, json optional)", Value: "raw"},
		// &cli.IntFlag{Name:         "max-restarts", Usage: "only restart the script COUNT times"},
		// &cli.IntFlag{Name:         "pid", Aliases: []string{"p"}, Usage: "specify pid file"},
		// &cli.StringFlag{Name:      "env", Usage: "specify which set of environment variables from ecosystem file must be injected"},
		// &cli.StringSliceFlag{Name: "filter-env", Usage: "filter out outgoing global values that contain provided strings"},
		// &cli.BoolFlag{Name:        "update-env", Aliases: []string{"a"}, Usage: "force an update of the environment with restart/reload (-a <=> apply)"},
		// &cli.BoolFlag{Name:        "execute-command", Aliases: []string{"x"}, Usage: "execute a program using fork system"},
		// &cli.DurationFlag{Name:    "exp-backoff-restart-delay", Usage: "specify a delay between restarts"},
		// &cli.IntFlag{Name:         "gid", Usage: "run target script with <gid> rights"},
		// &cli.IntFlag{Name:         "uid", Usage: "run target script with <uid> rights"},
		// &cli.StringSliceFlag{Name: "ignore-watch", Usage: "List of paths to ignore (name or regex)"},
		// &cli.BoolFlag{Name:        "no-autorestart", Usage: "start an app without automatic restart"},
		// &cli.DurationFlag{Name:    "restart-delay", Usage: "specify a delay between restarts"},
		// &cli.BoolFlag{Name:        "merge-logs", Usage: "merge logs from different instances but keep error and out separated"},
		// &cli.StringFlag{Name:      "interpreter", Usage: "set a specific interpreter to use for executing app", Value: "node"},
		// &cli.StringSliceFlag{Name: "interpreter-args", Usage: "set arguments to pass to the interpreter"},
		// &cli.BoolFlag{Name:        "fresh", Usage: "Rebuild Dockerfile"},
		// &cli.BoolFlag{Name:        "daemon", Usage: "Run container in Daemon mode (debug purposes)"},
		// &cli.BoolFlag{Name:        "container", Usage: "Start application in container mode"},
		// &cli.BoolFlag{Name:        "dist", Usage: "with --container; change local Dockerfile to containerize all files in current directory"},
		// &cli.StringFlag{Name:      "image-name", Usage: "with --dist; set the exported image name"},
		// &cli.BoolFlag{Name:        "dockerdaemon", Usage: "for debugging purpose"},
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
		return start(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.StringSlice("status"),
			ctx.Uint64Slice("id"),
		)
	},
}

func start(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters, statusFilters []string,
	idFilters []uint64,
) error {
	resp, err := db.New(_daemonDBFile).List()
	if err != nil {
		return err
	}

	procIDsToStart := internal.FilterProcs[uint64](
		resp,
		internal.WithAllIfNoFilters,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithStatuses(statusFilters),
		internal.WithTags(tagFilters),
	)

	client, deferFunc, err := NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(deferFunc)

	if _, err := client.Start(ctx, &pb.IDs{Ids: procIDsToStart}); err != nil {
		return fmt.Errorf("client.Start failed: %w", err)
	}

	fmt.Println(lo.ToAnySlice(procIDsToStart)...)

	return nil
}
