package internal

import "github.com/urfave/cli/v2"

func init() {
	AllCmds = append(AllCmds, StopCmd)
}

var StopCmd = &cli.Command{
	Name:      "stop",
	Usage:     "stop a process",
	ArgsUsage: "<id|name|namespace|all|json|stdin...>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "watch",
			Usage: "Stop watching folder for changes",
		},
	},
	Action: func(*cli.Context) error {
		//   .action(function(param) {
		//     forEachLimit(param, 1, function(script, next) {
		//       pm2.stop(script, next);
		//     }, function(err) {
		//       pm2.speedList(err ? 1 : 0);
		//     });
		return nil
	},
}
