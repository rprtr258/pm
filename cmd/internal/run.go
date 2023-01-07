package internal

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/db"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
)

func init() {
	AllCmds = append(AllCmds, RunCmd)
}

var RunCmd = &cli.Command{
	Name:      "run",
	ArgsUsage: "cmd args...",
	Usage:     "run new process and manage it",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Usage:    "set a name for the process",
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:    "tag",
			Aliases: []string{"t"},
			Usage:   "add specified tag",
		},
		&cli.StringFlag{
			Name:  "cwd",
			Usage: "set working directory",
		},
		&cli.StringFlag{
			Name:    "interpreter",
			Aliases: []string{"i"},
			Usage:   "set interpreter to executing command",
		},
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
		// &cli.StringSliceFlag{Name: "interpreter-args", Usage: "set arguments to pass to the interpreter"},
		// &cli.BoolFlag{Name:        "fresh", Usage: "Rebuild Dockerfile"},
		// &cli.BoolFlag{Name:        "daemon", Usage: "Run container in Daemon mode (debug purposes)"},
		// &cli.BoolFlag{Name:        "container", Usage: "Start application in container mode"},
		// &cli.BoolFlag{Name:        "dist", Usage: "with --container; change local Dockerfile to containerize all files in current directory"},
		// &cli.StringFlag{Name:      "image-name", Usage: "with --dist; set the exported image name"},
		// &cli.BoolFlag{Name:        "dockerdaemon", Usage: "for debugging purpose"},
	},
	Action: func(ctx *cli.Context) error {
		interpreter := ctx.String("interpreter")
		args := ctx.Args().Slice()

		var toRunArgs []string
		switch {
		case interpreter == "" && len(args) == 0:
			return errors.New("command expected")
		case interpreter != "" && len(args) != 1:
			return fmt.Errorf("interpreter %q set, thats why only single arg must be provided (but there were %d provided)", interpreter, len(args))
		case interpreter == "":
			toRunArgs = args
		case interpreter != "":
			toRunArgs = append(toRunArgs, strings.Split(interpreter, " ")...)
			toRunArgs = append(toRunArgs, args[0])
		default:
			return fmt.Errorf("unknown situation, interpreter=%q args=%v", interpreter, args)
		}

		var err error
		toRunArgs[0], err = exec.LookPath(toRunArgs[0])
		if err != nil {
			return fmt.Errorf("could not find executable %q: %w", toRunArgs[0], err)
		}

		name := ctx.String("name")
		return run(
			ctx.Context,
			toRunArgs,
			lo.If(name != "", internal.Valid(name)).
				Else(internal.Invalid[string]()),
			ctx.StringSlice("tag"),
		)
	},
}

func run(
	ctx context.Context,
	args []string,
	name internal.Optional[string],
	tags []string,
) error {
	if !name.Valid {
		return errors.New("name required") // TODO: remove
	}

	procData := db.ProcData{
		Status: db.Status{
			Status: db.StatusStarting,
		},
		Name:    name.Value,
		Cwd:     ".",
		Tags:    lo.Uniq(append(tags, "all")),
		Command: args[0], // TODO: clarify
		Args:    args[1:],
	}

	procID, err := db.New(_daemonDBFile).AddProc(procData)

	if err != nil {
		return err
	}

	procData.ID = procID

	procIDs := []uint64{uint64(procData.ID)}

	client, deferFunc, err := NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(deferFunc)

	if _, err := client.Start(ctx, &pb.IDs{Ids: procIDs}); err != nil {
		return fmt.Errorf("client.Start failed: %w", err)
	}

	fmt.Println(procData.ID)

	return nil
}
