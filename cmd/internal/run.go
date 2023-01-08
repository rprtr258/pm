package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	"go.uber.org/multierr"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
)

func init() {
	AllCmds = append(AllCmds, RunCmd)
}

type RunConfig struct {
	Name    internal.Optional[string]
	Command string
	Args    []string
	Tags    []string
	Cwd     internal.Optional[string]
}

func (cfg *RunConfig) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name    *string  `json:"name"`
		Cwd     *string  `json:"cwd"`
		Command string   `json:"command"`
		Args    []any    `json:"args"`
		Tags    []string `json:"tags"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	*cfg = RunConfig{
		Name:    internal.FromPtr(tmp.Name),
		Cwd:     internal.FromPtr(tmp.Cwd),
		Command: tmp.Command,
		Args: lo.Map(
			tmp.Args,
			func(elem any, _ int) string {
				return fmt.Sprint(elem)
			},
		),
		Tags: tmp.Tags,
	}

	return nil
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
		// TODO: interpreter args
		&cli.StringFlag{
			Name:      "config",
			Usage:     "config file to use",
			Aliases:   []string{"f"},
			TakesFile: true,
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
		if interpreter == "" {
			if ctx.IsSet("config") && isConfigFile(ctx.String("config")) {
				configs, err := loadConfig(ctx.String("config"))
				if err != nil {
					return err
				}

				return runConfigs(ctx.Context, configs, args)
			}

			if len(args) == 0 {
				return errors.New("command expected")
			}

			toRunArgs = args
		} else {
			if len(args) != 1 {
				return fmt.Errorf(
					"interpreter %q set, thats why only single arg must be provided (but there were %d provided)",
					interpreter,
					len(args),
				)
			}

			toRunArgs = append(toRunArgs, strings.Split(interpreter, " ")...)
			toRunArgs = append(toRunArgs, args[0])
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
		return run(ctx.Context, config)
	},
}

func loadConfig(filename string) ([]RunConfig, error) {
	vm := jsonnet.MakeVM()
	vm.ExtVar("now", time.Now().Format("15:04:05"))

	jsonText, err := vm.EvaluateFile(filename)
	if err != nil {
		return nil, err
	}

	var configs []RunConfig
	if err := json.Unmarshal([]byte(jsonText), &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

func isConfigFile(arg string) bool {
	stat, err := os.Stat(arg)
	if err != nil {
		return false
	}

	return !stat.IsDir()
}

func runConfigs(
	ctx context.Context,
	configs []RunConfig,
	names []string,
) error {
	if len(names) == 0 {
		var merr error
		for _, config := range configs {
			multierr.AppendInto(&merr, run(ctx, config))
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
		multierr.AppendInto(&merr, run(ctx, config))
	}
	return merr
}

func run(
	ctx context.Context,
	config RunConfig,
) error {
	client, err := client.NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(client.Close)

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
