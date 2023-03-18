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

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/xerr"
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
				Args: lo.Map(
					config.Args,
					func(arg any, _ int) string {
						if stringer, ok := arg.(fmt.Stringer); ok {
							return stringer.String()
						}
						// TODO: check arg types
						return fmt.Sprintf("%v", arg)
					},
				),
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
		return xerr.NewM("invalid config file",
			xerr.Field("configFilename", configFilename))
	}

	if !ctx.IsSet("config") {
		if err := cmd.Validate(ctx, nil); err != nil {
			return err
		}
	}

	client, err := client.NewGrpcClient()
	if err != nil {
		return xerr.NewWM(err, "new grpc client")
	}
	defer deferErr(client.Close)()

	list, err := client.List(ctx.Context)
	if err != nil {
		return xerr.NewWM(err, "server.list")
	}

	if !ctx.IsSet("config") {
		return cmd.Run(
			ctx,
			nil,
			client,
			list,
			list,
		)
	}

	configs, err := loadConfigs(configFilename)
	if err != nil {
		return err
	}

	if err := cmd.Validate(ctx, configs); err != nil {
		return err
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

	return cmd.Run(
		ctx,
		configs,
		client,
		list,
		configList,
	)
}

// { Name: "trigger", Usage:     "trigger process action", ArgsUsage: "<id|proc_name|namespace|all> <action_name> [params]", .action(function(pm_id, action_name, params) { pm2.trigger(pm_id, action_name, params); },

// { Name: "deploy", Usage:     "deploy your json", ArgsUsage: "<file|environment>", .action(function(cmd) { pm2.deploy(cmd, commander); },

// { Name: "id", commander.command('id <name>') .description('get process id by name') .action(function(name) { pm2.getProcessIdByName(name); }); },
// { Name: "pid", commander.command('[app_name]') .description('return pid of [app_name] or all') .action(function(app) { pm2.getPID(app); },
// { Name: "create", .description('return pid of [app_name] or all') .action(function() { pm2.boilerplate() },

// { Name: "startOrRestart", Usage:     "start or restart JSON file", ArgsUsage: "<json>", .action(function(file) { pm2._startJson(file, commander, 'restartProcessId'); },
// { Name: "startOrReload", Usage:     "start or gracefully reload JSON file", ArgsUsage: "<json>", .action(function(file) { pm2._startJson(file, commander, 'reloadProcessId'); },

// { Name: "restart", commander.command('restart <id|name|namespace|all|json|stdin...>') .option('--watch', 'Toggle watching folder for changes') .description('restart a process') .action(function(param) { Commander.js patch param = patchCommanderArg(param); let acc = [] forEachLimit(param, 1, function(script, next) { pm2.restart(script, commander, (err, apps) => { acc = acc.concat(apps) next(err) }); }, function(err) { pm2.speedList(err ? 1 : 0, acc); }); }); },
// { Name: "reload", commander.command('reload <id|name|namespace|all>') .description('reload processes (note that its for app using HTTP/HTTPS)') .action(function(pm2_id) { pm2.reload(pm2_id, commander); }); },
// { Name: "scale", commander.command('scale <app_name> <number>') .description('scale up/down a process in cluster mode depending on total_number param') .action(function(app_name, number) { pm2.scale(app_name, number); }); },

// { Name: "startOrGracefulReload", commander.command('startOrGracefulReload <json>') .description('start or gracefully reload JSON file') .action(function(file) { pm2._startJson(file, commander, 'reloadProcessId'); }); },

// { Name: "profile:mem", commander.command('profile:mem [time]') .description('Sample PM2 heap memory') .action(function(time) { pm2.profile('mem', time); }); },
// { Name: "profile:cpu", commander.command('profile:cpu [time]') .description('Profile PM2 cpu') .action(function(time) { pm2.profile('cpu', time); }); },

// { Name: "inspect", commander.command('inspect <name>') .description('inspect a process') .action(function(cmd) { pm2.inspect(cmd, commander); }); },

// { Name: "sendSignal", commander.command('sendSignal <signal> <pm2_id|name>') .description('send a system signal to the target process') .action(function(signal, pm2_id) { if (isNaN(parseInt(pm2_id))) { console.log(cst.PREFIX_MSG + 'Sending signal to process name ' + pm2_id); pm2.sendSignalToProcessName(signal, pm2_id); } else { console.log(cst.PREFIX_MSG + 'Sending signal to process id ' + pm2_id); pm2.sendSignalToProcessId(signal, pm2_id); } }); },

// { Name: "ping",
//   .description('ping pm2 daemon - if not up it will launch it')
//   .action(function() {
//     pm2.ping();
//   });
// },

// { Name: "updatePM2",
// commander.command('updatePM2')
//   .description('update in-memory PM2 with local PM2')
//   .action(function() {
//     pm2.update();
//   });
// commander.command('update')
//   .description('(alias) update in-memory PM2 with local PM2')
//   .action(function() {
//     pm2.update();
//   });
// },

// { Name: "install",
// Module specifics
// commander.command('install <module|git://url>')
//   .alias('module:install')
//   .option('--tarball', 'is local tarball')
//   .option('--install', 'run yarn install before starting module')
//   .option('--docker', 'is docker container')
//   .option('--v1', 'install module in v1 manner (do not use it)')
//   .option('--safe [time]', 'keep module backup, if new module fail = restore with previous')
//   .description('install or update a module and run it forever')
//   .action(function(plugin_name, opts) {
//     require('util')._extend(commander, opts);
//     pm2.install(plugin_name, commander);
//   });
// },

// { Name: "module:update",
// commander.command('module:update <module|git://url>')
//   .description('update a module and run it forever')
//   .action(function(plugin_name) {
//     pm2.install(plugin_name);
//   });
// },

// { Name: "module:generate",
// commander.command('module:generate [app_name]')
//   .description('Generate a sample module in current folder')
//   .action(function(app_name) {
//     pm2.generateModuleSample(app_name);
//   });
// },

// { Name: "uninstall",
// commander.command('uninstall <module>')
//   .alias('module:uninstall')
//   .description('stop and uninstall a module')
//   .action(function(plugin_name) {
//     pm2.uninstall(plugin_name);
//   });
// },

// { Name: "report",
//   .description('give a full pm2 report for https://github.com/Unitech/pm2/issues')
//   .action(function(key) {
//     pm2.report();
//   });
// },

// { Name: "link",
// PM2 I/O
// commander.command('link [secret] [public] [name]')
//   .option('--info-node [url]', 'set url info node')
//   .description('link with the pm2 monitoring dashboard')
//   .action(pm2.linkManagement.bind(pm2));
// },

// { Name: "unlink",
// commander.command('unlink')
//   .description('unlink with the pm2 monitoring dashboard')
//   .action(function() {
//     pm2.unlink();
//   });
// },

// { Name: "monitor",
// commander.command('monitor [name]')
//   .description('monitor target process')
//   .action(function(name) {
//     if (name == undefined) {
//       return plusHandler()
//     }
//     pm2.monitorState('monitor', name);
//   });
// },

// { Name: "unmonitor",
// commander.command('unmonitor [name]')
//   .description('unmonitor target process')
//   .action(function(name) {
//     pm2.monitorState('unmonitor', name);
//   });
// },

// { Name: "open",
//   .description('open the pm2 monitoring dashboard')
//   .action(function(name) {
//     pm2.openDashboard();
//   });
// },

// { Name: "plus",
// commander.command('plus [command] [option]')
//   .alias('register')
//   .option('--info-node [url]', 'set url info node for on-premise pm2 plus')
//   .option('-d --discrete', 'silent mode')
//   .option('-a --install-all', 'install all modules (force yes)')
//   .description('enable pm2 plus')
//   .action(plusHandler);
// function plusHandler (command, opts) {
//   if (opts && opts.infoNode) {
//     process.env.KEYMETRICS_NODE = opts.infoNode
//   }
//   return PM2ioHandler.launch(command, opts)
// }
// },

// { Name: "login",
//   .description('Login to pm2 plus')
//   .action(function() {
//     return plusHandler('login')
//   });
// },

// { Name: "logout",
//   .description('Logout from pm2 plus')
//   .action(function() {
//     return plusHandler('logout')
//   });
// },

// { Name: "dump",
//   .alias('save')
//   .option('--force', 'force deletion of dump file, even if empty')
//   .description('dump all processes for resurrecting them later')
//   .action(failOnUnknown(function(opts) {
//     pm2.dump(commander.force)
//   }));
// },

// { Name: "cleardump",
// Delete dump file
//   .description('Create empty dump file')
//   .action(failOnUnknown(function() {
//     pm2.clearDump();
//   }));
// },

// { Name: "resurrect",
//   .description('resurrect previously dumped processes')
//   .action(failOnUnknown(function() {
//     console.log(cst.PREFIX_MSG + 'Resurrecting');
//     pm2.resurrect();
//   }));
// },

// { Name: "send",
// commander.command('send <pm_id> <line>')
//   .description('send stdin to <pm_id>')
//   .action(function(pm_id, line) {
//     pm2.sendLineToStdin(pm_id, line);
//   });
// },

// { Name: "attach",
// Attach to stdin/stdout
// Not TTY ready
// commander.command('attach <pm_id> [command separator]')
//   .description('attach stdin/stdout to application identified by <pm_id>')
//   .action(function(pm_id, separator) {
//     pm2.attach(pm_id, separator);
//   });
// },

// { Name: "unstartup",
// commander.command('unstartup [platform]')
//   .description('disable the pm2 startup hook')
//   .action(function(platform) {
//     pm2.uninstallStartup(platform, commander);
//   });
// },

// { Name: "startup",
// commander.command('startup [platform]')
//   .description('enable the pm2 startup hook')
//   .action(function(platform) {
//     pm2.startup(platform, commander);
//   });
// },

// { Name: "logrotate",
//   .description('copy default logrotate configuration')
//   .action(function(cmd) {
//     pm2.logrotate(commander);
//   });
// },

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

// { Name: "reset",
// commander.command('reset <name|id|all>')
//   .description('reset counters for process')
//   .action(function(proc_id) {
//     pm2.reset(proc_id);
//   });
// },

// { Name: "describe",
// commander.command('describe <name|id>')
//   .description('describe all parameters of a process')
//   .alias("desc", "info", "show")
//   .action(function(proc_id) {
//     pm2.describe(proc_id);
//   });
// },

// { Name: "env",
// commander.command('env <id>')
//   .description('list all environment variables of a process id')
//   .action(function(proc_id) {
//     pm2.env(proc_id);
//   });
// },

// { Name: "sysmonit",
//   .description('start system monitoring daemon')
//   .action(function() {
//     pm2.launchSysMonitoring()
//   })
// },

// { Name: "monit",
// Dashboard command
// commander.command('')
//   .description('launch termcaps monitoring')
//   .action(function() {
//     pm2.dashboard();
//   });
// },

// { Name: "imonit",
//   .description('launch legacy termcaps monitoring')
//   .action(function() {
//     pm2.monit();
//   });
// },

// { Name: "dashboard",
//   .alias('dash')
//   .description('launch dashboard with monitoring and logs')
//   .action(function() {
//     pm2.dashboard();
//   });
// },

// { Name:      "flush",
// 	Usage:     "flush logs",
// 	ArgsUsage: "[api]",
// .action(function(api) {
//   pm2.flush(api);
//  }
// },

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
// 	},
//   .action(function(id, cmd) {
//     var Logs = require('../API/Log.js');
//     if (!id) id = 'all';
//     var line = 15;
//     var raw  = false;
//     var exclusive = false;
//     var timestamp = false;
//     var highlight = false;
//     if(!isNaN(parseInt(cmd.lines))) {
//       line = parseInt(cmd.lines);
//     }
//     if (cmd.parent.rawArgs.indexOf('--raw') !== -1)
//       raw = true;
//     if (cmd.timestamp)
//       timestamp = typeof cmd.timestamp == 'string' ? cmd.timestamp : 'YYYY-MM-DD-HH:mm:ss';
//     if (cmd.highlight)
//       highlight = typeof cmd.highlight == 'string' ? cmd.highlight : false;
//     if (cmd.out == true)
//       exclusive = 'out';
//     if (cmd.err == true)
//       exclusive = 'err';
//     if (cmd.nostream == true)
//       pm2.printLogs(id, line, raw, timestamp, exclusive);
//     else if (cmd.json == true)
//       Logs.jsonStream(pm2.Client, id);
//     else if (cmd.format == true)
//       Logs.formatStream(pm2.Client, id, false, 'YYYY-MM-DD-HH:mm:ssZZ', exclusive, highlight);
//     else
//       pm2.streamLogs(id, line, raw, timestamp, exclusive, highlight);
// },

// { Name:  "kill",
// 	Usage: "kill daemon",
//   .action(failOnUnknown(function(arg) {
//     pm2.killDaemon(function() {
//       process.exit(cst.SUCCESS_EXIT);
// },

// { Name:      "pull",
// 	Usage:     "updates repository for a given app",
// 	ArgsUsage: "<name> [commit_id]",
//   .action(function(pm2_name, commit_id) {
//     if (commit_id !== undefined) {
//       pm2._pullCommitId({
//         pm2_name: pm2_name,
//         commit_id: commit_id
//       });
//     }
//     else
//       pm2.pullAndRestart(pm2_name);
// },}

// { Name:      "forward",
// 	Usage:     "updates repository to the next commit for a given app",
// 	ArgsUsage: "<name>",
//   .action(function(pm2_name) {
//     pm2.forward(pm2_name);
// },}

// { Name:      "backward",
// 	Usage:     "downgrades repository to the previous commit for a given app",
// 	ArgsUsage: "<name>",
//   .action(function(pm2_name) {
//     pm2.backward(pm2_name);
// },}

// { Name:  "deepUpdate",
// 	Usage: "performs a deep update of PM2",
//     pm2.deepUpdate();
// },

// { Name:      "serve",
// 	Usage:     "serve a path over http",
// 	ArgsUsage: "[path] [port]",
// 	Aliases:   []string{"expose"},
// 	Flags:     []cli.Flag{
//   .option('--port [port]', 'specify port to listen to')
//   .option('--spa', 'always serving index.html on inexistant sub path')
//   .option('--basic-auth-username [username]', 'set basic auth username')
//   .option('--basic-auth-password [password]', 'set basic auth password')
//   .option('--monitor [frontend-app]', 'frontend app monitoring (auto integrate snippet on html files)')
// 	},
//   .action(function (path, port, cmd) {
//     pm2.serve(path, port || cmd.port, cmd, commander);
// },}
