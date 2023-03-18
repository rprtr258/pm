package internal

import (
	"fmt"
	"log"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/client"
	pm_daemon "github.com/rprtr258/pm/internal/daemon"
	"github.com/rprtr258/pm/internal/go-daemon"
)

func init() {
	AllCmds = append(AllCmds, DaemonCmd)
}

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "manage daemon",
	Subcommands: []*cli.Command{
		{
			Name:    "start",
			Aliases: []string{"restart"},
			Usage:   "launch daemon process",
			Action: func(ctx *cli.Context) error {
				return daemonStart()
			},
		},
		{
			Name:    "stop",
			Aliases: []string{"kill"},
			Usage:   "stop daemon process",
			Action: func(ctx *cli.Context) error {
				return daemonStop()
			},
		},
		{
			Name:  "run",
			Usage: "run daemon, DON'T USE BY HAND IF YOU DON'T KNOW WHAT YOU ARE DOING",
			Action: func(ctx *cli.Context) error {
				return daemonRun()
			},
		},
		{
			Name:    "status",
			Usage:   "check daemon status",
			Aliases: []string{"ps"},
			Action: func(ctx *cli.Context) error {
				client, err := client.NewGrpcClient()
				if err != nil {
					return err
				}

				// TODO: print daemon process info

				if err := client.HealthCheck(ctx.Context); err != nil {
					return err
				}

				fmt.Println("ok")

				return nil
			},
		},
	},
}

func daemonStart() error {
	// TODO: move to internal
	daemonCtx := &daemon.Context{
		PidFileName: internal.FileDaemonPid,
		PidFilePerm: 0o644,
		LogFileName: internal.FileDaemonLog,
		LogFilePerm: 0o640,
		WorkDir:     "./",
		Umask:       0o27,
		Args:        []string{"pm", "daemon", "start"},
		Chroot:      "",
		Env:         nil,
		Credential:  nil,
	}

	if err := pm_daemon.Kill(daemonCtx, internal.SocketDaemonRPC); err != nil {
		return xerr.NewWM(err, "kill daemon process")
	}

	daemonProcess, err := daemonCtx.Reborn()
	if err != nil {
		return xerr.NewWM(err, "reborn daemon")
	}

	if daemonProcess != nil {
		fmt.Println(daemonProcess.Pid)
		if err := daemonCtx.Release(); err != nil {
			return xerr.NewWM(err, "daemon release")
		}
		return nil
	}

	defer deferErr(daemonCtx.Release)()

	return daemonRun()
}

func daemonStop() error {
	daemonCtx := &daemon.Context{
		PidFileName: internal.FileDaemonPid,
		PidFilePerm: 0o644,
		LogFileName: internal.FileDaemonLog,
		LogFilePerm: 0o640,
		WorkDir:     "./",
		Umask:       0o27,
		Args:        []string{"pm", "daemon", "start"},
		Chroot:      "",
		Env:         nil,
		Credential:  nil,
	}

	return pm_daemon.Kill(daemonCtx, internal.SocketDaemonRPC)
}

func daemonRun() error {
	return pm_daemon.Run(internal.SocketDaemonRPC, internal.FileDaemonDBDir, internal.DirHome)
}

func deferErr(closer func() error) func() {
	return func() {
		if err := closer(); err != nil {
			log.Println("some defer action failed:", err)
		}
	}
}
