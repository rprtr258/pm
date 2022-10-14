package main

import (
	"fmt"
	"os"
	"path"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	cst "github.com/rprtr258/pm"
)

// processes - proc name -> pid
var (
	processes map[string]int = make(map[string]int)
	userHome                 = os.Getenv("HOME")
	homeDir                  = path.Join(userHome, ".pm")
)

func main() {
	startCmd := &cli.Command{
		Name:      "start",
		ArgsUsage: "<cmd args...|<name|namespace|file|ecosystem|id>...>",
		Usage:     "start and daemonize an app",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "watch", Usage: "Watch folder for changes"},
			&cli.BoolFlag{Name: "fresh", Usage: "Rebuild Dockerfile"},
			&cli.BoolFlag{Name: "fresh", Usage: "Rebuild Dockerfile"},
			&cli.BoolFlag{Name: "daemon", Usage: "Run container in Daemon mode (debug purposes)"},
			&cli.BoolFlag{Name: "container", Usage: "Start application in container mode"},
			&cli.BoolFlag{Name: "dist", Usage: "with --container; change local Dockerfile to containerize all files in current directory"},
			&cli.StringFlag{Name: "image-name", Usage: "with --dist; set the exported image name"},
			&cli.BoolFlag{Name: "node-version", Usage: "with --container, set a specific major Node.js version"},
			&cli.BoolFlag{Name: "dockerdaemon", Usage: "for debugging purpose"},
		},
		Action: func(ctx *cli.Context) error {
			// if (opts.container == true && opts.dist == true)
			//   return pm2.dockerMode(cmd, opts, 'distribution');
			// else if (opts.container == true)
			//   return pm2.dockerMode(cmd, opts, 'development');

			// if (cmd == "-") {
			//   process.stdin.resume();
			//   process.stdin.setEncoding('utf8');
			//   process.stdin.on('data', function (cmd) {
			//     process.stdin.pause();
			//     pm2._startJson(cmd, commander, 'restartProcessId', 'pipe');
			//   });
			// }
			// else {
			//   // Commander.js patch
			//   cmd = patchCommanderArg(cmd);
			//   if (cmd.length === 0) {
			//     cmd = [cst.APP_CONF_DEFAULT_FILE];
			//   }
			//   let acc = []
			//   forEachLimit(cmd, 1, function(script, next) {
			//     pm2.start(script, commander, (err, apps) => {
			//       acc = acc.concat(apps)
			//       next(err)
			//     });
			//   }, function(err, dt) {
			//     if (err && err.message &&
			//         (err.message.includes('Script not found') === true ||
			//          err.message.includes('NOT AVAILABLE IN PATH') === true)) {
			//       pm2.exitCli(1)
			//     }
			//     else
			//       pm2.speedList(err ? 1 : 0, acc);
			//   });
			// }

			// ==================

			// args := ctx.Args().Slice()
			// if len(args) < 1 {
			// 	return errors.New("command expected")
			// }

			// if err := os.Mkdir(path.Join(homeDir, name), 0755); err != nil {
			// 	return err
			// }

			// stdoutLogFile, err := os.OpenFile(path.Join(homeDir, name, "stdout"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			// if err != nil {
			// 	return err
			// }

			// stderrLogFile, err := os.OpenFile(path.Join(homeDir, name, "stderr"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			// if err != nil {
			// 	return err
			// }

			// pidFile, err := os.OpenFile(path.Join(homeDir, name, "pid"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			// if err != nil {
			// 	return err
			// }

			// // TODO: syscall.ForkExec()
			// cmd := exec.CommandContext(ctx.Context, args[0], args[1:]...)
			// cmd.Stdout = stdoutLogFile
			// cmd.Stderr = stderrLogFile
			// if err := cmd.Start(); err != nil {
			// 	return err
			// }

			// if _, err := pidFile.WriteString(strconv.Itoa(cmd.Process.Pid)); err != nil {
			// 	return err
			// }

			// processes[name] = cmd.Process.Pid

			return nil
		},
	}

	listCmd := &cli.Command{
		Name:    "list",
		Aliases: []string{"l"},
		Action: func(*cli.Context) error {
			fs, err := os.ReadDir(homeDir)
			if err != nil {
				return err
			}

			for _, f := range fs {
				if !f.IsDir() {
					fmt.Fprintf(os.Stderr, "found strange file %q which should not exist\n", path.Join(homeDir, f.Name()))
					continue
				}

				fmt.Printf("%#v", f.Name())
			}
			return nil
		},
	}

	stopCmd := &cli.Command{
		Name:      "stop",
		Aliases:   []string{"kill", "s"},
		ArgsUsage: "<name>",
		Action: func(*cli.Context) error {
			return nil
		},
	}

	examplesCmd := &cli.Command{
		Name:  "examples",
		Usage: "display pm2 usage examples",
		Action: func(ctx *cli.Context) error {
			_, err := fmt.Printf(`%s%s
- Start and add a process to the pm2 process list:
%s

- Show the process list:
%s

- Stop and delete a process from the pm2 process list:
%s

- Stop, start and restart a process from the process list:
%s
%s
%s

- Clusterize an app to all CPU cores available:
%s

- Update pm2:
%s

- Install pm2 auto completion:
%s
`,
				cst.PREFIX_MSG,
				color.HiBlackString("pm2 usage examples:"),
				color.CyanString("pm2 start app.js --name app"),
				color.CyanString("pm2 ls"),
				color.CyanString("pm2 delete app"),
				color.CyanString("pm2 stop app"),
				color.CyanString("pm2 start app"),
				color.CyanString("pm2 restart app"),
				color.CyanString("pm2 start -i max"),
				color.CyanString("npm install pm2 -g && pm2 update"),
				color.CyanString("pm2 completion install"),
			)
			return err
		},
	}

	app := &cli.App{
		Name:  "pm",
		Usage: "manage running processes",
		Flags: []cli.Flag{
			// version(pkg.version)
			&cli.BoolFlag{Name: "version", Aliases: []string{"v"}, Usage: "print pm2 version"},
			&cli.BoolFlag{Name: "silent", Aliases: []string{"s"}, Usage: "hide all messages", Value: false},
			&cli.StringSliceFlag{Name: "ext", Usage: "watch only this file extensions"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "set a name for the process in the process list"},
			&cli.BoolFlag{Name: "mini-list", Aliases: []string{"m"}, Usage: "display a compacted list without formatting"},
			&cli.StringFlag{Name: "interpreter", Usage: "set a specific interpreter to use for executing app", Value: "node"},
			&cli.StringSliceFlag{Name: "interpreter-args", Usage: "set arguments to pass to the interpreter"},
			&cli.BoolFlag{Name: "output", Aliases: []string{"o"}, Usage: "specify log file for stdout"},
			&cli.PathFlag{Name: "error", Aliases: []string{"e"}, Usage: "specify log file for stderr"},
			&cli.PathFlag{Name: "log", Aliases: []string{"l"}, Usage: "specify log file which gathers both stdout and stderr"},
			&cli.StringSliceFlag{Name: "filter-env", Usage: "filter out outgoing global values that contain provided strings"},
			&cli.StringFlag{Name: "log-type", Usage: "specify log output style (raw by default, json optional)", Value: "raw"},
			&cli.StringFlag{Name: "log-date-format", Usage: "add custom prefix timestamp to logs"},
			&cli.BoolFlag{Name: "time", Usage: "enable time logging"},
			&cli.BoolFlag{Name: "disable-logs", Usage: "disable all logs storage"},
			&cli.StringFlag{Name: "env", Usage: "specify which set of environment variables from ecosystem file must be injected"},
			&cli.BoolFlag{Name: "update-env", Aliases: []string{"a"}, Usage: "force an update of the environment with restart/reload (-a <=> apply)"},
			&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "force actions"},
			&cli.BoolFlag{Name: "instances <number>", Aliases: []string{"i"}, Usage: "launch [number] instances (for networked app)(load balanced)"},
			&cli.IntFlag{Name: "parallel", Usage: "number of parallel actions (for restart/reload)"},
			&cli.BoolFlag{Name: "shutdown-with-message", Usage: "shutdown an application with process.send('shutdown') instead of process.kill(pid, SIGINT)"},
			&cli.IntFlag{Name: "pid", Aliases: []string{"p"}, Usage: "specify pid file"},
			&cli.DurationFlag{Name: "kill-timeout", Aliases: []string{"k"}, Usage: "delay before sending final SIGKILL signal to process"},
			&cli.DurationFlag{Name: "listen-timeout", Usage: "listen timeout on application reload"},
			&cli.IntFlag{Name: "max-memory-restart", Usage: "Restart the app if an amount of memory is exceeded (in bytes)"},
			&cli.DurationFlag{Name: "restart-delay", Usage: "specify a delay between restarts"},
			&cli.DurationFlag{Name: "exp-backoff-restart-delay", Usage: "specify a delay between restarts"},
			&cli.BoolFlag{Name: "execute-command", Aliases: []string{"x"}, Usage: "execute a program using fork system"},
			&cli.IntFlag{Name: "max-restarts", Usage: "only restart the script COUNT times"},
			&cli.StringFlag{Name: "user", Aliases: []string{"u"}, Usage: "define user when generating startup script"},
			&cli.IntFlag{Name: "uid", Usage: "run target script with <uid> rights"},
			&cli.IntFlag{Name: "gid", Usage: "run target script with <gid> rights"},
			&cli.StringFlag{Name: "namespace", Usage: "start application within specified namespace"},
			&cli.StringFlag{Name: "cwd", Usage: "run target script from path <cwd>"},
			&cli.StringFlag{Name: "home-path", Usage: "define home path when generating startup script"},
			&cli.BoolFlag{Name: "wait-ip", Usage: "override systemd script to wait for full internet connectivity to launch pm2"},
			&cli.BoolFlag{Name: "service-name", Usage: "define service name when generating startup script"},
			&cli.StringFlag{Name: "cron-restart", Aliases: []string{"c", "cron"}, Usage: "restart a running process based on a cron pattern"},
			&cli.BoolFlag{Name: "write", Aliases: []string{"w"}, Usage: "write configuration in local folder"},
			&cli.BoolFlag{Name: "no-daemon", Usage: "run pm2 daemon in the foreground if it doesn\t exist already"},
			&cli.BoolFlag{Name: "source-map-support", Usage: "force source map support"},
			&cli.BoolFlag{Name: "only", Usage: "with json declaration, allow to only act on one application"},
			&cli.BoolFlag{Name: "disable-source-map-support", Usage: "force source map support"},
			&cli.BoolFlag{Name: "wait-ready", Usage: "ask pm2 to wait for ready event from your app"},
			&cli.BoolFlag{Name: "merge-logs", Usage: "merge logs from different instances but keep error and out separated"},
			&cli.StringSliceFlag{Name: "watch", Usage: "watch application folder for changes"},
			&cli.StringSliceFlag{Name: "ignore-watch", Usage: "List of paths to ignore (name or regex)"},
			&cli.DurationFlag{Name: "watch-delay", Usage: "specify a restart delay after changing files (--watch-delay 4 (in sec) or 4000ms)"},
			&cli.BoolFlag{Name: "no-color", Usage: "skip colors"},
			&cli.BoolFlag{Name: "no-vizion", Usage: "start an app without vizion feature (versioning control)"},
			&cli.BoolFlag{Name: "no-autorestart", Usage: "start an app without automatic restart"},
			&cli.BoolFlag{Name: "no-treekill", Usage: "Only kill the main process, not detached children"},
			&cli.BoolFlag{Name: "no-pmx", Usage: "start an app without pmx"},
			&cli.BoolFlag{Name: "no-automation", Usage: "start an app without pmx"},
			&cli.BoolFlag{Name: "trace", Usage: "enable transaction tracing with km"},
			&cli.BoolFlag{Name: "disable-trace", Usage: "disable transaction tracing with km"},
			&cli.BoolFlag{Name: "sort", Usage: "sort <field_name:sort> sort process according to field's name"},
			&cli.BoolFlag{Name: "attach", Usage: "attach logging after your start/restart/stop/reload"},
			&cli.BoolFlag{Name: "v8", Usage: "enable v8 data collecting"},
			&cli.BoolFlag{Name: "event-loop-inspector", Usage: "enable event-loop-inspector dump in pmx"},
			&cli.BoolFlag{Name: "deep-monitoring", Usage: "enable all monitoring tools (equivalent to --v8 --event-loop-inspector --trace)"},
		},
		Commands: []*cli.Command{
			startCmd,
			stopCmd,
			listCmd,
			examplesCmd,
			&cli.Command{
				Name:      "trigger",
				Usage:     "trigger process action",
				ArgsUsage: "<id|proc_name|namespace|all> <action_name> [params]",
				//   .action(function(pm_id, action_name, params) {
				//     pm2.trigger(pm_id, action_name, params);
			},
			&cli.Command{
				Name:      "deploy",
				Usage:     "deploy your json",
				ArgsUsage: "<file|environment>",
				//   .action(function(cmd) {
				//     pm2.deploy(cmd, commander);
			},
			&cli.Command{
				Name:      "startOrRestart",
				Usage:     "start or restart JSON file",
				ArgsUsage: "<json>",
				//   .action(function(file) {
				//     pm2._startJson(file, commander, 'restartProcessId');
			},
			&cli.Command{
				Name:      "startOrReload",
				Usage:     "start or gracefully reload JSON file",
				ArgsUsage: "<json>",
				//   .action(function(file) {
				//     pm2._startJson(file, commander, 'reloadProcessId');
			},
			&cli.Command{
				Name: "pid",
				// commander.command('[app_name]')
				//   .description('return pid of [app_name] or all')
				//   .action(function(app) {
				//     pm2.getPID(app);
			},
			&cli.Command{
				Name: "create",
				//   .description('return pid of [app_name] or all')
				//   .action(function() {
				//     pm2.boilerplate()
			},
			&cli.Command{
				Name: "startOrGracefulReload",
				// commander.command('startOrGracefulReload <json>')
				//   .description('start or gracefully reload JSON file')
				//   .action(function(file) {
				//     pm2._startJson(file, commander, 'reloadProcessId');
				//   });
			},
			&cli.Command{
				Name: "stop",
				// commander.command('stop <id|name|namespace|all|json|stdin...>')
				//   .option('--watch', 'Stop watching folder for changes')
				//   .description('stop a process')
				//   .action(function(param) {
				//     forEachLimit(param, 1, function(script, next) {
				//       pm2.stop(script, next);
				//     }, function(err) {
				//       pm2.speedList(err ? 1 : 0);
				//     });
				//   });
			},
			&cli.Command{
				Name: "restart",
				// commander.command('restart <id|name|namespace|all|json|stdin...>')
				//   .option('--watch', 'Toggle watching folder for changes')
				//   .description('restart a process')
				//   .action(function(param) {
				//     // Commander.js patch
				//     param = patchCommanderArg(param);
				//     let acc = []
				//     forEachLimit(param, 1, function(script, next) {
				//       pm2.restart(script, commander, (err, apps) => {
				//         acc = acc.concat(apps)
				//         next(err)
				//       });
				//     }, function(err) {
				//       pm2.speedList(err ? 1 : 0, acc);
				//     });
				//   });
			},
			&cli.Command{
				Name: "scale",
				// commander.command('scale <app_name> <number>')
				//   .description('scale up/down a process in cluster mode depending on total_number param')
				//   .action(function(app_name, number) {
				//     pm2.scale(app_name, number);
				//   });
			},
			&cli.Command{
				Name: "profile:mem",
				// commander.command('profile:mem [time]')
				//   .description('Sample PM2 heap memory')
				//   .action(function(time) {
				//     pm2.profile('mem', time);
				//   });
			},
			&cli.Command{
				Name: "profile:cpu",
				// commander.command('profile:cpu [time]')
				//   .description('Profile PM2 cpu')
				//   .action(function(time) {
				//     pm2.profile('cpu', time);
				//   });
			},
			&cli.Command{
				Name: "reload",
				// commander.command('reload <id|name|namespace|all>')
				//   .description('reload processes (note that its for app using HTTP/HTTPS)')
				//   .action(function(pm2_id) {
				//     pm2.reload(pm2_id, commander);
				//   });
			},
			&cli.Command{
				Name: "id",
				// commander.command('id <name>')
				//   .description('get process id by name')
				//   .action(function(name) {
				//     pm2.getProcessIdByName(name);
				//   });
			},
			&cli.Command{
				Name: "inspect",
				// commander.command('inspect <name>')
				//   .description('inspect a process')
				//   .action(function(cmd) {
				//     pm2.inspect(cmd, commander);
				//   });
			},
			&cli.Command{
				Name: "delete",
				// commander.command('delete <name|id|namespace|script|all|json|stdin...>')
				//   .alias('del')
				//   .description('stop and delete a process from pm2 process list')
				//   .action(function(name) {
				//     if (name == "-") {
				//       process.stdin.resume();
				//       process.stdin.setEncoding('utf8');
				//       process.stdin.on('data', function (param) {
				//         process.stdin.pause();
				//         pm2.delete(param, 'pipe');
				//       });
				//     } else
				//       forEachLimit(name, 1, function(script, next) {
				//         pm2.delete(script,'', next);
				//       }, function(err) {
				//         pm2.speedList(err ? 1 : 0);
				//       });
				//   });
			},
			&cli.Command{
				Name: "sendSignal",
				// commander.command('sendSignal <signal> <pm2_id|name>')
				//   .description('send a system signal to the target process')
				//   .action(function(signal, pm2_id) {
				//     if (isNaN(parseInt(pm2_id))) {
				//       console.log(cst.PREFIX_MSG + 'Sending signal to process name ' + pm2_id);
				//       pm2.sendSignalToProcessName(signal, pm2_id);
				//     } else {
				//       console.log(cst.PREFIX_MSG + 'Sending signal to process id ' + pm2_id);
				//       pm2.sendSignalToProcessId(signal, pm2_id);
				//     }
				//   });
			},
			&cli.Command{
				Name: "ping",
				//   .description('ping pm2 daemon - if not up it will launch it')
				//   .action(function() {
				//     pm2.ping();
				//   });
			},
			&cli.Command{
				Name: "updatePM2",
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
			},
			&cli.Command{
				Name: "install",
				// // Module specifics
				// commander.command('install <module|git:// url>')
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
			},
			&cli.Command{
				Name: "module:update",
				// commander.command('module:update <module|git:// url>')
				//   .description('update a module and run it forever')
				//   .action(function(plugin_name) {
				//     pm2.install(plugin_name);
				//   });
			},
			&cli.Command{
				Name: "module:generate",
				// commander.command('module:generate [app_name]')
				//   .description('Generate a sample module in current folder')
				//   .action(function(app_name) {
				//     pm2.generateModuleSample(app_name);
				//   });
			},
			&cli.Command{
				Name: "uninstall",
				// commander.command('uninstall <module>')
				//   .alias('module:uninstall')
				//   .description('stop and uninstall a module')
				//   .action(function(plugin_name) {
				//     pm2.uninstall(plugin_name);
				//   });
			},
			&cli.Command{
				Name: "set",
				// commander.command('set [key] [value]')
				//   .description('sets the specified config <key> <value>')
				//   .action(function(key, value) {
				//     pm2.set(key, value);
				//   });
			},
			&cli.Command{
				Name: "multiset",
				// commander.command('multiset <value>')
				//   .description('multiset eg "key1 val1 key2 val2')
				//   .action(function(str) {
				//     pm2.multiset(str);
				//   });
			},
			&cli.Command{
				Name: "get",
				// commander.command('get [key]')
				//   .description('get value for <key>')
				//   .action(function(key) {
				//     pm2.get(key);
				//   });
			},
			&cli.Command{
				Name: "conf",
				// commander.command('conf [key] [value]')
				//   .description('get / set module config values')
				//   .action(function(key, value) {
				//     pm2.get()
				//   });
			},
			&cli.Command{
				Name: "config",
				// commander.command('config <key> [value]')
				//   .description('get / set module config values')
				//   .action(function(key, value) {
				//     pm2.conf(key, value);
				//   });
			},
			&cli.Command{
				Name: "unset",
				// commander.command('unset <key>')
				//   .description('clears the specified config <key>')
				//   .action(function(key) {
				//     pm2.unset(key);
				//   });
			},
			&cli.Command{
				Name: "report",
				//   .description('give a full pm2 report for https://github.com/Unitech/pm2/issues')
				//   .action(function(key) {
				//     pm2.report();
				//   });
			},
			&cli.Command{
				Name: "link",
				// // PM2 I/O
				// commander.command('link [secret] [public] [name]')
				//   .option('--info-node [url]', 'set url info node')
				//   .description('link with the pm2 monitoring dashboard')
				//   .action(pm2.linkManagement.bind(pm2));
			},
			&cli.Command{
				Name: "unlink",
				// commander.command('unlink')
				//   .description('unlink with the pm2 monitoring dashboard')
				//   .action(function() {
				//     pm2.unlink();
				//   });
			},
			&cli.Command{
				Name: "monitor",
				// commander.command('monitor [name]')
				//   .description('monitor target process')
				//   .action(function(name) {
				//     if (name === undefined) {
				//       return plusHandler()
				//     }
				//     pm2.monitorState('monitor', name);
				//   });
			},
			&cli.Command{
				Name: "unmonitor",
				// commander.command('unmonitor [name]')
				//   .description('unmonitor target process')
				//   .action(function(name) {
				//     pm2.monitorState('unmonitor', name);
				//   });
			},
			&cli.Command{
				Name: "open",
				//   .description('open the pm2 monitoring dashboard')
				//   .action(function(name) {
				//     pm2.openDashboard();
				//   });
			},
			&cli.Command{
				Name: "plus",
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
			},
			&cli.Command{
				Name: "login",
				//   .description('Login to pm2 plus')
				//   .action(function() {
				//     return plusHandler('login')
				//   });
			},
			&cli.Command{
				Name: "logout",
				//   .description('Logout from pm2 plus')
				//   .action(function() {
				//     return plusHandler('logout')
				//   });
			},
			&cli.Command{
				Name: "dump",
				//   .alias('save')
				//   .option('--force', 'force deletion of dump file, even if empty')
				//   .description('dump all processes for resurrecting them later')
				//   .action(failOnUnknown(function(opts) {
				//     pm2.dump(commander.force)
				//   }));
			},
			&cli.Command{
				Name: "cleardump",
				// // Delete dump file
				//   .description('Create empty dump file')
				//   .action(failOnUnknown(function() {
				//     pm2.clearDump();
				//   }));
			},
			&cli.Command{
				Name: "send",
				// commander.command('send <pm_id> <line>')
				//   .description('send stdin to <pm_id>')
				//   .action(function(pm_id, line) {
				//     pm2.sendLineToStdin(pm_id, line);
				//   });
			},
			&cli.Command{
				Name: "attach",
				// // Attach to stdin/stdout
				// // Not TTY ready
				// commander.command('attach <pm_id> [command separator]')
				//   .description('attach stdin/stdout to application identified by <pm_id>')
				//   .action(function(pm_id, separator) {
				//     pm2.attach(pm_id, separator);
				//   });
			},
			&cli.Command{
				Name: "resurrect",
				//   .description('resurrect previously dumped processes')
				//   .action(failOnUnknown(function() {
				//     console.log(cst.PREFIX_MSG + 'Resurrecting');
				//     pm2.resurrect();
				//   }));
			},
			&cli.Command{
				Name: "unstartup",
				// commander.command('unstartup [platform]')
				//   .description('disable the pm2 startup hook')
				//   .action(function(platform) {
				//     pm2.uninstallStartup(platform, commander);
				//   });
			},
			&cli.Command{
				Name: "startup",
				// commander.command('startup [platform]')
				//   .description('enable the pm2 startup hook')
				//   .action(function(platform) {
				//     pm2.startup(platform, commander);
				//   });
			},
			&cli.Command{
				Name: "logrotate",
				//   .description('copy default logrotate configuration')
				//   .action(function(cmd) {
				//     pm2.logrotate(commander);
				//   });
			},
			&cli.Command{
				Name: "ecosystem",
				// // Sample generate
				// commander.command('ecosystem [mode]')
				//   .alias('init')
				//   .description('generate a process conf file. (mode = null or simple)')
				//   .action(function(mode) {
				//     pm2.generateSample(mode);
				//   });
			},
			&cli.Command{
				Name: "reset",
				// commander.command('reset <name|id|all>')
				//   .description('reset counters for process')
				//   .action(function(proc_id) {
				//     pm2.reset(proc_id);
				//   });
			},
			&cli.Command{
				Name: "describe",
				// commander.command('describe <name|id>')
				//   .description('describe all parameters of a process')
				//   .action(function(proc_id) {
				//     pm2.describe(proc_id);
				//   });
			},
			&cli.Command{
				Name: "desc",
				// commander.command('desc <name|id>')
				//   .description('(alias) describe all parameters of a process')
				//   .action(function(proc_id) {
				//     pm2.describe(proc_id);
				//   });
			},
			&cli.Command{
				Name: "info",
				// commander.command('info <name|id>')
				//   .description('(alias) describe all parameters of a process')
				//   .action(function(proc_id) {
				//     pm2.describe(proc_id);
				//   });
			},
			&cli.Command{
				Name: "show",
				// commander.command('show <name|id>')
				//   .description('(alias) describe all parameters of a process')
				//   .action(function(proc_id) {
				//     pm2.describe(proc_id);
				//   });
			},
			&cli.Command{
				Name: "env",
				// commander.command('env <id>')
				//   .description('list all environment variables of a process id')
				//   .action(function(proc_id) {
				//     pm2.env(proc_id);
				//   });
			},
			&cli.Command{
				Name:    "list",
				Aliases: []string{"ls", "l", "ps", "status"},
				//   .description('list all processes')
				//   .action(function() {
				//     pm2.list(commander)
				//   });
			},
			&cli.Command{
				Name: "jlist",
				//   .description('list all processes in JSON format')
				//   .action(function() {
				//     pm2.jlist()
				//   });
			},
			&cli.Command{
				Name: "sysmonit",
				//   .description('start system monitoring daemon')
				//   .action(function() {
				//     pm2.launchSysMonitoring()
				//   })
			},
			&cli.Command{
				Name: "slist",
				//   .alias('sysinfos')
				//   .option('-t --tree', 'show as tree')
				//   .description('list system infos in JSON')
				//   .action(function(opts) {
				//     pm2.slist(opts.tree)
				//   })
			},
			&cli.Command{
				Name: "prettylist",
				//   .description('print json in a prettified JSON')
				//   .action(failOnUnknown(function() {
				//     pm2.jlist(true);
				//   }));
			},
			&cli.Command{
				Name: "monit",
				// // Dashboard command
				// commander.command('')
				//   .description('launch termcaps monitoring')
				//   .action(function() {
				//     pm2.dashboard();
				//   });
			},
			&cli.Command{
				Name: "imonit",
				//   .description('launch legacy termcaps monitoring')
				//   .action(function() {
				//     pm2.monit();
				//   });
			},
			&cli.Command{
				Name: "dashboard",
				//   .alias('dash')
				//   .description('launch dashboard with monitoring and logs')
				//   .action(function() {
				//     pm2.dashboard();
				//   });
			},
			&cli.Command{
				Name:      "flush",
				Usage:     "flush logs",
				ArgsUsage: "[api]",
				// .action(function(api) {
				//   pm2.flush(api);
			},
			&cli.Command{
				Name:  "reloadLogs",
				Usage: "reload all logs",
				//     pm2.reloadLogs();
			},
			&cli.Command{
				Name:      "logs",
				Usage:     "stream logs file. Default stream all logs",
				ArgsUsage: "[id|name|namespace]",
				Flags:     []cli.Flag{
					//   .option('--json', 'json log output')
					//   .option('--format', 'formated log output')
					//   .option('--raw', 'raw output')
					//   .option('--err', 'only shows error output')
					//   .option('--out', 'only shows standard output')
					//   .option('--lines <n>', 'output the last N lines, instead of the last 15 by default')
					//   .option('--timestamp [format]', 'add timestamps (default format YYYY-MM-DD-HH:mm:ss)')
					//   .option('--nostream', 'print logs without lauching the log stream')
					//   .option('--highlight [value]', 'highlights the given value')
				},
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
				//       timestamp = typeof cmd.timestamp === 'string' ? cmd.timestamp : 'YYYY-MM-DD-HH:mm:ss';

				//     if (cmd.highlight)
				//       highlight = typeof cmd.highlight === 'string' ? cmd.highlight : false;

				//     if (cmd.out === true)
				//       exclusive = 'out';

				//     if (cmd.err === true)
				//       exclusive = 'err';

				//     if (cmd.nostream === true)
				//       pm2.printLogs(id, line, raw, timestamp, exclusive);
				//     else if (cmd.json === true)
				//       Logs.jsonStream(pm2.Client, id);
				//     else if (cmd.format === true)
				//       Logs.formatStream(pm2.Client, id, false, 'YYYY-MM-DD-HH:mm:ssZZ', exclusive, highlight);
				//     else
				//       pm2.streamLogs(id, line, raw, timestamp, exclusive, highlight);
			},
			&cli.Command{
				Name:  "kill",
				Usage: "kill daemon",
				//   .action(failOnUnknown(function(arg) {
				//     pm2.killDaemon(function() {
				//       process.exit(cst.SUCCESS_EXIT);
			},
			&cli.Command{
				Name:      "pull",
				Usage:     "updates repository for a given app",
				ArgsUsage: "<name> [commit_id]",
				//   .action(function(pm2_name, commit_id) {
				//     if (commit_id !== undefined) {
				//       pm2._pullCommitId({
				//         pm2_name: pm2_name,
				//         commit_id: commit_id
				//       });
				//     }
				//     else
				//       pm2.pullAndRestart(pm2_name);
			},
			&cli.Command{
				Name:      "forward",
				Usage:     "updates repository to the next commit for a given app",
				ArgsUsage: "<name>",
				//   .action(function(pm2_name) {
				//     pm2.forward(pm2_name);
			},
			&cli.Command{
				Name:      "backward",
				Usage:     "downgrades repository to the previous commit for a given app",
				ArgsUsage: "<name>",
				//   .action(function(pm2_name) {
				//     pm2.backward(pm2_name);
			},
			&cli.Command{
				Name:  "deepUpdate",
				Usage: "performs a deep update of PM2",
				//     pm2.deepUpdate();
			},
			&cli.Command{
				Name:      "serve",
				Usage:     "serve a path over http",
				ArgsUsage: "[path] [port]",
				Aliases:   []string{"expose"},
				Flags:     []cli.Flag{
					//   .option('--port [port]', 'specify port to listen to')
					//   .option('--spa', 'always serving index.html on inexistant sub path')
					//   .option('--basic-auth-username [username]', 'set basic auth username')
					//   .option('--basic-auth-password [password]', 'set basic auth password')
					//   .option('--monitor [frontend-app]', 'frontend app monitoring (auto integrate snippet on html files)')
				},
				//   .action(function (path, port, cmd) {
				//     pm2.serve(path, port || cmd.port, cmd, commander);
			},
			&cli.Command{
				Name: "autoinstall",
				//     pm2.autoinstall()
			},
		},
		Before: func(*cli.Context) error {
			if _, err := os.Stat(homeDir); os.IsNotExist(err) {
				os.Mkdir(homeDir, 0755)
			}

			return nil
		},
	}

	//   pm2.getVersion(function(err, remote_version) {
	//     if (!err && (pkg.version != remote_version)) {
	//       console.log('');
	//       console.log(chalk.red.bold('>>>> In-memory PM2 is out-of-date, do:\n>>>> $ pm2 update'));
	//       console.log('In memory PM2 version:', chalk.blue.bold(remote_version));
	//       console.log('Local PM2 version:', chalk.blue.bold(pkg.version));
	//       console.log('');
	//     }
	//   });

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
