package internal

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	"go.uber.org/multierr"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
)

func init() {
	AllCmds = append(
		AllCmds,
		&cli.Command{
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
				// TODO: interpreter args
				configFlag,
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
				return executeProcCommand(
					ctx,
					&runCmd{
						// TODO: change to destinations
						name:        ctx.String("name"),
						args:        ctx.Args().Slice(),
						tags:        ctx.StringSlice("tag"),
						cwd:         ctx.String("pwd"),
						interpreter: ctx.String("interpreter"),
					},
				)
			},
		},
	)
}

type runCmd struct {
	name        string
	args        []string
	tags        []string
	cwd         string
	interpreter string
}

func (cmd *runCmd) Validate(ctx *cli.Context, configs []RunConfig) error {
	// TODO: validate params
	return nil
}

func (cmd *runCmd) Run(
	ctx *cli.Context,
	configs []RunConfig,
	client client.Client,
	list db.DB,
	configList db.DB,
) error {
	var toRunArgs []string
	if cmd.interpreter == "" {
		if configs != nil {
			return runConfigs(ctx.Context, cmd.args, configs, client)
		}

		if len(cmd.args) == 0 {
			return errors.New("command expected")
		}

		toRunArgs = cmd.args
	} else {
		if configs != nil {
			return fmt.Errorf("either interpreter with cmd or config must be provided, not both")
		}

		if len(cmd.args) != 1 {
			return fmt.Errorf(
				"interpreter %q set, thats why only single arg must be provided (but there were %d provided)",
				cmd.interpreter,
				len(cmd.args),
			)
		}

		toRunArgs = append(toRunArgs, strings.Split(cmd.interpreter, " ")...)
		toRunArgs = append(toRunArgs, cmd.args[0])
	}

	var err error
	if err != nil {
		return fmt.Errorf("could not find executable %q: %w", toRunArgs[0], err)
	}

	name := ctx.String("name")
	config := RunConfig{
		Command: toRunArgs[0],
		Args:    toRunArgs[1:],
		Name: lo.If(name != "", internal.Valid(name)).
			Else(internal.Invalid[string]()),
		Tags: ctx.StringSlice("tag"),
	}
	return run(ctx.Context, config, client)

}

func runConfigs(
	ctx context.Context,
	names []string,
	configs []RunConfig,
	client client.Client,
) error {
	if len(names) == 0 {
		var merr error
		for _, config := range configs {
			multierr.AppendInto(&merr, run(ctx, config, client))
		}
		return merr
	}

	configsByName := make(map[string]RunConfig, len(names))
	for _, cfg := range configs {
		if !cfg.Name.Valid || !lo.Contains(names, cfg.Name.Value) {
			continue
		}

		configsByName[cfg.Name.Value] = cfg
	}

	var merr error
	for _, name := range names {
		if _, ok := configsByName[name]; !ok {
			multierr.AppendInto(&merr, fmt.Errorf("unknown name of proc: %s", name))
		}
	}
	if merr != nil {
		return merr
	}

	for _, config := range configsByName {
		multierr.AppendInto(&merr, run(ctx, config, client))
	}
	return merr
}

func run(
	ctx context.Context,
	config RunConfig,
	client client.Client,
) error {
	command, err := exec.LookPath(config.Command)
	if err != nil {
		return err
	}

	procID, err := client.Create(ctx, &api.ProcessOptions{
		Command: command,
		Args:    config.Args,
		Name:    config.Name.Ptr(),
		Cwd:     lo.ToPtr(internal.OrDefault(".", config.Cwd)),
		Tags:    config.Tags,
	})
	if err != nil {
		return err
	}

	if err := client.Start(ctx, []uint64{procID}); err != nil {
		return fmt.Errorf("client.Start failed: %w", err)
	}

	fmt.Println(procID)

	return nil
}
