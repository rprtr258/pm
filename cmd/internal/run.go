package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/rprtr258/pm/api"
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
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferFunc()

		name := ctx.String(_flagName)

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("command expected")
		}

		resp, err := client.SayHello(context.TODO(), &api.HelloRequest{
			Name: fmt.Sprint(name, args),
		})
		if err != nil {
			return err
		}

		fmt.Println("got from server", resp.GetMessage())
		return nil

		// ==================
		// if (cmd == "-") {
		//   process.stdin.resume();
		//   process.stdin.setEncoding('utf8');
		//   process.stdin.on('data', function (cmd) {
		//     process.stdin.pause();
		//     pm2._startJson(cmd, commander, 'restartProcessId', 'pipe');
		//   });
		// } else {
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
		//     } else
		//       pm2.speedList(err ? 1 : 0, acc);
		//   });
		// }
	},
}
