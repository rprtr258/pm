package internal

import (
	"fmt"

	"github.com/sevlyar/go-daemon"
	"github.com/urfave/cli/v2"

	pm_daemon "github.com/rprtr258/pm/internal/daemon"
)

func init() {
	AllCmds = append(AllCmds, DaemonCmd)
}

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "manage daemon",
	Subcommands: []*cli.Command{
		{
			Name:  "start",
			Usage: "launch daemon process",
			Action: func(ctx *cli.Context) error {
				daemonCtx := &daemon.Context{
					PidFileName: _daemonPidFile,
					PidFilePerm: 0644,
					LogFileName: _daemonLogFile,
					LogFilePerm: 0640,
					WorkDir:     "./",
					Umask:       027,
					Args:        []string{"pm", "daemon", "start"},
				}

				pm_daemon.Kill(daemonCtx, _daemonRpcSocket)

				d, err := daemonCtx.Reborn()
				if err != nil {
					return fmt.Errorf("reborn failed: %w", err)
				}

				if d != nil {
					fmt.Println(d.Pid)
					return nil
				}

				defer daemonCtx.Release()

				return pm_daemon.Run(_daemonRpcSocket, _daemonDBFile)
			},
		},
		{
			Name:  "stop",
			Usage: "stop daemon process",
			Action: func(ctx *cli.Context) error {
				daemonCtx := &daemon.Context{
					PidFileName: _daemonPidFile,
					PidFilePerm: 0644,
					LogFileName: _daemonLogFile,
					LogFilePerm: 0640,
					WorkDir:     "./",
					Umask:       027,
					Args:        []string{"pm", "daemon", "start"},
				}

				return pm_daemon.Kill(daemonCtx, _daemonRpcSocket)
			},
		},
		{
			Name:  "run",
			Usage: "run daemon, DON'T USE BY HAND IF YOU DON'T KNOW WHAT YOU ARE DOING",
			Action: func(ctx *cli.Context) error {
				return pm_daemon.Run(_daemonRpcSocket, _daemonDBFile)
			},
		},
	},
}
