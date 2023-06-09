package cli

import (
	"fmt"
	"log"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	internal "github.com/rprtr258/pm/internal/core"
	pm_daemon "github.com/rprtr258/pm/internal/core/daemon"
	"github.com/rprtr258/pm/internal/infra/go-daemon"
	"github.com/rprtr258/pm/pkg/client"
)

var _daemonCmd = &cli.Command{
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
				client, errNewClient := client.NewGrpcClient()
				if errNewClient != nil {
					return xerr.NewWM(errNewClient, "create grpc client")
				}

				// TODO: print daemon process info

				if errHealth := client.HealthCheck(ctx.Context); errHealth != nil {
					return xerr.NewWM(errHealth, "check daemon health")
				}

				fmt.Println("ok")

				return nil
			},
		},
	},
}

func newDaemonContext() *daemon.Context {
	return &daemon.Context{
		PidFileName: internal.FileDaemonPid,
		PidFilePerm: 0o644, //nolint:gomnd // default pid file permissions, rwxr--r--
		LogFileName: internal.FileDaemonLog,
		LogFilePerm: 0o640, //nolint:gomnd // default log file permissions, rwxr-----
		WorkDir:     "./",
		Umask:       0o27, //nolint:gomnd // don't know
		Args:        []string{"pm", "daemon", "start"},
		Chroot:      "",
		Env:         nil,
		Credential:  nil,
	}
}

func daemonStart() error {
	// TODO: move to internal
	daemonCtx := newDaemonContext()

	if errKill := pm_daemon.Kill(daemonCtx, internal.SocketDaemonRPC); errKill != nil {
		return xerr.NewWM(errKill, "kill daemon process")
	}

	daemonProcess, errReborn := daemonCtx.Reborn()
	if errReborn != nil {
		return xerr.NewWM(errReborn, "reborn daemon")
	}

	if daemonProcess != nil { // parent
		fmt.Println(daemonProcess.Pid)
		if err := daemonCtx.Release(); err != nil {
			return xerr.NewWM(err, "daemon release")
		}
		return nil
	}

	// child/daemon
	defer deferErr(daemonCtx.Release)()

	return daemonRun()
}

func daemonStop() error {
	daemonCtx := newDaemonContext()
	if errKill := pm_daemon.Kill(daemonCtx, internal.SocketDaemonRPC); errKill != nil {
		return xerr.NewWM(errKill, "kill daemon process")
	}

	return nil
}

func daemonRun() error {
	if errRun := pm_daemon.Run(internal.SocketDaemonRPC, internal.FileDaemonDBDir, internal.DirHome); errRun != nil {
		return xerr.NewWM(errRun, "run daemon")
	}

	return nil
}

func deferErr(closer func() error) func() {
	return func() {
		if err := closer(); err != nil {
			log.Println("some defer action failed:", err)
		}
	}
}
