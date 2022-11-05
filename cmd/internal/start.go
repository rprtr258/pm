package internal

import (
	"errors"
	"fmt"
	"strings"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/daemon"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
)

func init() {
	AllCmds = append(AllCmds, StartCmd)
}

var StartCmd = &cli.Command{
	Name: "start",
	// ArgsUsage: "<'cmd args...'|name|namespace|config|id>...",
	//     oneof filter {
	//         Tags tags = 10; // all procs having all of those tags
	//         string name = 11; // proc with such name
	//         google.protobuf.Empty all = 12; // all
	//         string config = 13; // all procs described in config
	//     };
	ArgsUsage: "cmd args...",
	Usage:     "start and daemonize an app",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Usage:    "set a name for the process",
			Required: false, // TODO: gen name if not provided
		},
		&cli.StringSliceFlag{
			Name:  "tags",
			Usage: "assign specified tags",
		},
		&cli.StringFlag{
			Name:  "cwd",
			Usage: "set working directory",
		},
		// TODO: script + names..?
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
		// &cli.BoolFlag{Name:        "instances <number>", Aliases: []string{"i"}, Usage: "launch [number] instances (for networked app)(load balanced)"}, // TODO: remove
		// &cli.BoolFlag{Name:        "merge-logs", Usage: "merge logs from different instances but keep error and out separated"},
		// &cli.StringFlag{Name:      "interpreter", Usage: "set a specific interpreter to use for executing app", Value: "node"}, // TODO: remove
		// &cli.StringSliceFlag{Name: "interpreter-args", Usage: "set arguments to pass to the interpreter"}, // TODO: remove
		// &cli.BoolFlag{Name:        "fresh", Usage: "Rebuild Dockerfile"},
		// &cli.BoolFlag{Name:        "daemon", Usage: "Run container in Daemon mode (debug purposes)"},
		// &cli.BoolFlag{Name:        "container", Usage: "Start application in container mode"},
		// &cli.BoolFlag{Name:        "dist", Usage: "with --container; change local Dockerfile to containerize all files in current directory"},
		// &cli.StringFlag{Name:      "image-name", Usage: "with --dist; set the exported image name"},
		// &cli.BoolFlag{Name:        "node-version", Usage: "with --container, set a specific major Node.js version"},
		// &cli.BoolFlag{Name:        "dockerdaemon", Usage: "for debugging purpose"},
	},
	Action: func(ctx *cli.Context) error {
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferFunc()

		db, err := daemon.New(_daemonDBFile)
		if err != nil {
			return err
		}
		defer db.Close()

		name := ctx.String("name")

		// TODO: do more smartly
		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("command expected")
		}

		procData := daemon.ProcData{
			Status: daemon.Status{
				Status: daemon.StatusStarting,
			},
			Name: name,
			Cwd:  ".",
			Tags: lo.Uniq(append(ctx.StringSlice("tags"), "all")),
			Cmd:  strings.Join(args, " "),
		}

		procID, err := db.AddProc(procData)
		if err != nil {
			return err
		}

		procData.ID = procID

		if _, err := client.Start(ctx.Context, &pb.IDs{Ids: []uint64{uint64(procData.ID)}}); err != nil {
			return err
		}

		fmt.Println(procData.ID)
		return nil
	},
}
