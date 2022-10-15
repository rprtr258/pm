package internal

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/urfave/cli/v2"
)

const (
	_flagName = "name"
)

var StartCmd = &cli.Command{
	Name: "start",
	// ArgsUsage: "<'cmd args...'|name|namespace|config|id>...",
	ArgsUsage: "cmd args...",
	Usage:     "start and daemonize an app",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: _flagName, Aliases: []string{"n"}, Usage: "set a name for the process"},
		// &cli.BoolFlag{Name: "watch", Usage: "Watch folder for changes"},
		// &cli.BoolFlag{Name: "fresh", Usage: "Rebuild Dockerfile"},
		// &cli.BoolFlag{Name: "daemon", Usage: "Run container in Daemon mode (debug purposes)"},
		// &cli.BoolFlag{Name: "container", Usage: "Start application in container mode"},
		// &cli.BoolFlag{Name: "dist", Usage: "with --container; change local Dockerfile to containerize all files in current directory"},
		// &cli.StringFlag{Name: "image-name", Usage: "with --dist; set the exported image name"},
		// &cli.BoolFlag{Name: "node-version", Usage: "with --container, set a specific major Node.js version"},
		// &cli.BoolFlag{Name: "dockerdaemon", Usage: "for debugging purpose"},
	},
	Action: func(ctx *cli.Context) error {
		name := ctx.String(_flagName)

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("command expected")
		}

		if err := os.Mkdir(path.Join(HomeDir, name), 0755); err != nil {
			return err
		}

		stdoutLogFile, err := os.OpenFile(path.Join(HomeDir, name, "stdout"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		stderrLogFile, err := os.OpenFile(path.Join(HomeDir, name, "stderr"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		pidFile, err := os.OpenFile(path.Join(HomeDir, name, "pid"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}

		// TODO: syscall.ForkExec()
		cmd := exec.CommandContext(ctx.Context, args[0], args[1:]...)
		cmd.Stdout = stdoutLogFile
		cmd.Stderr = stderrLogFile
		if err := cmd.Start(); err != nil {
			return err
		}

		if _, err := pidFile.WriteString(strconv.Itoa(cmd.Process.Pid)); err != nil {
			return err
		}

		Processes[name] = cmd.Process.Pid

		return nil

		// ==================
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
		//   if (cmd.length == 0) {
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
		//         (err.message.includes('Script not found') == true ||
		//          err.message.includes('NOT AVAILABLE IN PATH') == true)) {
		//       pm2.exitCli(1)
		//     }
		//     else
		//       pm2.speedList(err ? 1 : 0, acc);
		//   });
		// }
	},
}
