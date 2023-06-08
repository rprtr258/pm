package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
)

type RunConfig struct {
	Args    []string
	Tags    []string
	Command string
	Cwd     string
	Name    internal.Optional[string]
}

func (cfg *RunConfig) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name    *string  `json:"name"`
		Cwd     string   `json:"cwd"`
		Command string   `json:"command"`
		Args    []any    `json:"args"`
		Tags    []string `json:"tags"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return xerr.NewWM(err, "json.unmarshal")
	}

	*cfg = RunConfig{
		Name:    internal.FromPtr(tmp.Name),
		Cwd:     tmp.Cwd,
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

// procCommand is any command changning procs state
// e.g. start, stop, delete, etc.
type procCommand interface {
	// Validate input parameters. Returns error if invalid parameters were found.
	// configs is nill if no config file provided.
	Validate(ctx *cli.Context, configs []RunConfig) error
	// Run command given all the input data.
	Run(
		ctx *cli.Context,
		configs []RunConfig,
		client client.Client,
		list map[db.ProcID]db.ProcData,
		configList map[db.ProcID]db.ProcData,
	) error
}

var (
	AllCmds []*cli.Command

	configFlag = &cli.StringFlag{
		Name:      "config",
		Usage:     "config file to use",
		Aliases:   []string{"f"},
		TakesFile: true,
	}
)

func isConfigFile(arg string) bool {
	stat, err := os.Stat(arg)
	if err != nil {
		return false
	}

	return !stat.IsDir()
}

func loadConfigs(filename string) ([]RunConfig, error) {
	vm := jsonnet.MakeVM()
	vm.ExtVar("now", time.Now().Format("15:04:05"))

	jsonText, err := vm.EvaluateFile(filename)
	if err != nil {
		return nil, xerr.NewWM(err, "evaluate jsonnet file")
	}

	type configScanDTO struct {
		Name    *string
		Cwd     *string
		Command string
		Args    []any
		Tags    []string
	}
	var scannedConfigs []configScanDTO
	if err := json.Unmarshal([]byte(jsonText), &scannedConfigs); err != nil {
		return nil, xerr.NewWM(err, "json.unmarshal")
	}

	return lo.Map(
		scannedConfigs,
		func(config configScanDTO, _ int) RunConfig {
			cwd := config.Cwd
			if cwd == nil {
				cwd = lo.ToPtr(filepath.Dir(filename))
			}

			return RunConfig{
				Name:    internal.FromPtr(config.Name),
				Command: config.Command,
				Args: lo.Map(config.Args, func(arg any, i int) string { //nolint:varnamelen // i is index
					switch a := arg.(type) {
					case fmt.Stringer:
						return a.String()
					case int, int8, int16, int32, int64,
						uint, uint8, uint16, uint32, uint64,
						float32, float64, bool, string:
						return fmt.Sprint(arg)
					default:
						argStr := fmt.Sprintf("%v", arg)
						log.Errorf("unknown arg type", log.F{
							"arg":    argStr,
							"i":      i,
							"config": config,
						})

						return argStr
					}
				}),
				Tags: config.Tags,
				Cwd:  *cwd,
			}
		},
	), nil
}

func executeProcCommand(
	ctx *cli.Context,
	cmd procCommand,
) error {
	// TODO: *string destination
	configFilename := ctx.String("config")

	if ctx.IsSet("config") && !isConfigFile(configFilename) {
		return xerr.NewM("invalid config file", xerr.Fields{"configFilename": configFilename})
	}

	if !ctx.IsSet("config") {
		if err := cmd.Validate(ctx, nil); err != nil {
			return xerr.NewWM(err, "validate config")
		}
	}

	client, errList := client.NewGrpcClient()
	if errList != nil {
		return xerr.NewWM(errList, "new grpc client")
	}
	defer deferErr(client.Close)()

	list, errList := client.List(ctx.Context)
	if errList != nil {
		return xerr.NewWM(errList, "server.list")
	}

	if !ctx.IsSet("config") {
		if errRun := cmd.Run(
			ctx,
			nil,
			client,
			list,
			list,
		); errRun != nil {
			return xerr.NewWM(errRun, "run")
		}
	}

	configs, errLoadConfigs := loadConfigs(configFilename)
	if errLoadConfigs != nil {
		return errLoadConfigs
	}

	if err := cmd.Validate(ctx, configs); err != nil {
		return xerr.NewWM(err, "validate run configs")
	}

	names := lo.FilterMap(
		configs,
		func(cfg RunConfig, _ int) (string, bool) {
			return cfg.Name.Value, cfg.Name.Valid
		},
	)

	configList := lo.PickBy(
		list,
		func(_ db.ProcID, procData db.ProcData) bool {
			return lo.Contains(names, procData.Name)
		},
	)

	if errRun := cmd.Run(
		ctx,
		configs,
		client,
		list,
		configList,
	); errRun != nil {
		return xerr.NewWM(errRun, "run config list")
	}

	return nil
}

// { Name: "pid", commander.command('[app_name]')
// .description('return pid of [app_name] or all') .action(function(app) { pm2.getPID(app); },

// Name: "restart", commander.command('restart <id|name|namespace|all|json|stdin...>') .description('restart a process')

// { Name: "inspect",
// commander.command('inspect <name|id>')
//   .description('inspect process')
//   .alias("desc", "info", "show")
//   .action(function(proc_id) {
//     pm2.describe(proc_id);
//   });
// },

// { Name: "sendSignal", commander.command('sendSignal <signal> <pm2_id|name>')
// .description('send a system signal to the target process')

// { Name: "ping", .description('ping pm2 daemon - if not up it will launch it')

// { Name: "update",
//   .description('update in-memory PM2 with local PM2')
//   .action(function() {
//     pm2.update();
//   });

// { Name: "report",
//   .description('give a full pm2 report for https://github.com/Unitech/pm2/issues')
//   .action(function(key) {
//     pm2.report();
//   });
// },

// { Name: "link", PM2 I/O
// commander.command('link [secret] [public] [name]')
//   .option('--info-node [url]', 'set url info node')
//   .description('link with the pm2 monitoring dashboard')

// { Name: "unlink", commander.command('unlink')
//   .description('unlink with the pm2 monitoring dashboard')

// { Name: "monitor",
// commander.command('monitor [name]')
//   .description('monitor target process / open monitoring dashboard')

// { Name: "unmonitor",
// commander.command('unmonitor [name]')
//   .description('unmonitor target process')

// { Name: "dump",
//   .alias('save')
//   .option('--force', 'force deletion of dump file, even if empty')
//   .option('--clear', 'empty dump file')
//   .description('dump all processes for resurrecting them later')
//   .action(failOnUnknown(function(opts) {
//     pm2.dump(commander.force)
//   }));
// },

// { Name: "resurrect",
//   .description('resurrect previously dumped processes')

// { Name: "send", commander.command('send <pm_id> <line>') .description('send stdin to <pm_id>')

// { Name: "attach", Attach to stdin/stdout
// commander.command('attach <pm_id> [command separator]')
//   .description('attach stdin/stdout to application identified by <pm_id>')

// { Name: "startup", commander.command('startup [platform]') .description('enable the pm2 startup hook')
// { Name: "unstartup", commander.command('unstartup') .description('disable the pm2 startup hook')

// { Name: "ecosystem",
// Sample generate
// commander.command('ecosystem [mode]')
//   .alias('init')
//   .description('generate a process conf file. (mode = null or simple)')
//   .action(function(mode) {
//     pm2.generateSample(mode);
//   });
// },
// &cli.BoolFlag{Name:        "service-name", Usage: "define service name when generating startup script"},
// &cli.StringFlag{Name:      "home-path", Usage: "define home path when generating startup script"},
// &cli.StringFlag{Name:      "user", Aliases: []string{"u"}, Usage: "define user when generating startup script"},
// &cli.BoolFlag{Name:        "write", Aliases: []string{"w"}, Usage: "write configuration in local folder"},

// { Name: "env",
// commander.command('env <id>')
//   .description('list all environment variables of a process id')
//   .action(function(proc_id) {
//     pm2.env(proc_id);
//   });
// },

// { Name: "monit",
// Dashboard command
//   .alias('dash', "dashboard")
//   .description('launch termcaps monitoring')
//   .description('launch dashboard with monitoring and logs')

// { Name: "imonit",
//   .description('launch legacy termcaps monitoring')
//   .action(function() {
//     pm2.monit();
//   });
// },

// { Name:      "flush",
// 	Usage:     "flush logs",
// 	ArgsUsage: "[api]",

// { Name:  "reloadLogs",
// 	Usage: "reload all logs",
//     pm2.reloadLogs();
// },

// { Name:      "logs",
// 	Usage:     "stream logs file. Default stream all logs",
// 	ArgsUsage: "[id|name|namespace]",
// 	Flags:     []cli.Flag{
//   .option('--json', 'json log output')
//   .option('--format', 'formated log output')
//   .option('--raw', 'raw output')
//   .option('--err', 'only shows error output')
//   .option('--out', 'only shows standard output')
//   .option('--lines <n>', 'output the last N lines, instead of the last 15 by default')
//   .option('--timestamp [format]', 'add timestamps (default format YYYY-MM-DD-HH:mm:ss)')
//   .option('--nostream', 'print logs without lauching the log stream')
//   .option('--highlight [value]', 'highlights the given value')

// { Name:      "serve",
// 	Usage:     "serve a path over http",
// 	ArgsUsage: "[path] [port]",
// 	Aliases:   []string{"expose"},
// 	Flags:     []cli.Flag{
//   .option('--port [port]', 'specify port to listen to')
//   .option('--spa', 'always serving index.html on inexistant sub path')
//   .option('--basic-auth-username [username]', 'set basic auth username')
//   .option('--basic-auth-password [password]', 'set basic auth password')
