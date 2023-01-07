package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	pm_daemon "github.com/rprtr258/pm/internal/daemon"
	"github.com/rprtr258/pm/internal/go-daemon"
)

var (
	_userHome        = os.Getenv("HOME")
	HomeDir          = filepath.Join(_userHome, ".pm")
	_daemonPidFile   = filepath.Join(HomeDir, "pm.pid")
	_daemonLogFile   = filepath.Join(HomeDir, "pm.log")
	_daemonRpcSocket = filepath.Join(HomeDir, "rpc.sock")
	_daemonLogsDir   = filepath.Join(HomeDir, "logs")
	_daemonDBFile    = filepath.Join(HomeDir, "pm.db")
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
	},
}

func daemonStart() error {
	// TODO: move to internal
	daemonCtx := &daemon.Context{
		PidFileName: _daemonPidFile,
		PidFilePerm: 0644,
		LogFileName: _daemonLogFile,
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"pm", "daemon", "start"},
	}

	if err := pm_daemon.Kill(daemonCtx, _daemonRpcSocket); err != nil {
		return fmt.Errorf("killing daemon failed: %w", err)
	}

	d, err := daemonCtx.Reborn()
	if err != nil {
		return fmt.Errorf("reborn daemon failed: %w", err)
	}

	if d != nil {
		fmt.Println(d.Pid)
		return daemonCtx.Release()
	}

	defer deferErr(daemonCtx.Release)

	return pm_daemon.Run(_daemonRpcSocket, _daemonDBFile, HomeDir)
}

func daemonStop() error {
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
}

func daemonRun() error {
	return pm_daemon.Run(_daemonRpcSocket, _daemonDBFile, HomeDir)
}

func deferErr(close func() error) {
	if err := close(); err != nil {
		log.Println("some defer action failed:", err)
	}
}
