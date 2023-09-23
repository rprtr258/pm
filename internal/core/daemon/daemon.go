package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"google.golang.org/grpc"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/pm"
	log2 "github.com/rprtr258/pm/internal/infra/cli/log"
	"github.com/rprtr258/pm/internal/infra/cli/log/buffer"
	"github.com/rprtr258/pm/internal/infra/go-daemon"
	"github.com/rprtr258/pm/pkg/client"
)

var (
	_dirProcsLogs = filepath.Join(core.DirHome, "logs")
	_filePid      = filepath.Join(core.DirHome, "pm.pid")
	_fileLog      = filepath.Join(core.DirHome, "pm.log")
	_dirDB        = filepath.Join(core.DirHome, "db")
)

func Status(ctx context.Context) error {
	pmDaemon, errNewClient := client.New()
	if errNewClient != nil {
		return xerr.NewWM(errNewClient, "create grpc client")
	}

	app, errNewApp := pm.New(pmDaemon)
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "create app")
	}

	status, errHealth := app.CheckDaemon(ctx)
	if errHealth != nil {
		return xerr.NewWM(errHealth, "check daemon")
	}

	// highlight special chars
	for k, v := range status.Envs {
		if !strings.ContainsAny(v, "\n\r\t ") {
			status.Envs[k] = strings.NewReplacer(
				"\n", buffer.String(`\n`, buffer.FgGreen),
				"\r", buffer.String(`\r`, buffer.FgGreen),
				"\t", buffer.String(`\t`, buffer.FgGreen),
				" ", buffer.String(`\x20`, buffer.FgGreen),
			).Replace(v)
		}
	}

	// crop long values
	for k, v := range status.Envs {
		if len(v) <= 100 {
			continue
		}

		status.Envs[k] = v[:50] + buffer.String("...", buffer.FgBlue) + v[len(v)-50:]
	}

	log2.Info().
		Any("Args", status.Args).
		Any("Envs", status.Envs).
		Str("Executable", status.Executable).
		Any("CWD", status.Cwd).
		Any("Groups", status.Groups).
		Any("Page Size:", status.PageSize).
		Any("Hostname", status.Hostname).
		Any("User Cache Dir", status.UserCacheDir).
		Any("User Config Dir", status.UserConfigDir).
		Any("User Home Dir", status.UserHomeDir).
		Any("PID", status.PID).
		Any("PPID", status.PPID).
		Any("UID", status.UID).
		Any("EUID", status.EUID).
		Any("GID", status.GID).
		Any("EGID", status.EGID).
		Msg("Daemon info")

	return nil
}

// TODO: move to daemon infra
var _daemonCtx = &daemon.Context{
	PidFileName: _filePid,
	PidFilePerm: 0o644, //nolint:gomnd // default pid file permissions, rwxr--r--
	LogFileName: _fileLog,
	LogFilePerm: 0o640, //nolint:gomnd // default log file permissions, rwxr-----
	WorkDir:     "./",
	Umask:       0o27, //nolint:gomnd // don't know
	Args:        []string{"pm", "daemon", "start"},
	Chroot:      "",
	Env:         nil,
	Credential:  nil,
}

// Kill daemon. If daemon is already killed, do nothing.
func Kill() error {
	if err := os.Remove(core.SocketRPC); err != nil && !errors.Is(err, os.ErrNotExist) {
		return xerr.NewWM(err, "remove socket file")
	}

	proc, err := _daemonCtx.Search()
	if err != nil {
		if err == daemon.ErrDaemonNotFound {
			log.Info().Msg("daemon already killed or did not exist")
			return nil
		}

		return xerr.NewWM(err, "search daemon")
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		if err == os.ErrProcessDone {
			log.Info().Msg("daemon is done while killing")
			return nil
		}

		return xerr.NewWM(err, "kill daemon process")
	}

	doneCh := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			<-ticker.C
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				// process is dead, err is ignored
				doneCh <- struct{}{}
				ticker.Stop()
			}
		}
	}()

	select {
	case <-doneCh:
	case <-time.After(5 * time.Second): //nolint:gomnd // arbitrary timeout
		if err := proc.Kill(); err != nil {
			if err == os.ErrProcessDone {
				log.Info().Msg("daemon is done while killing")
				return nil
			}

			return xerr.NewWM(err, "kill daemon process")
		}
	}

	return nil
}

func startDaemon(ctx context.Context) (int, error) {
	proc, errReborn := _daemonCtx.Reborn()
	if errReborn != nil {
		return 0, xerr.NewWM(errReborn, "reborn daemon")
	}
	defer func() {
		if err := _daemonCtx.Release(); err != nil {
			log.Error().
				Err(err).
				Msg("release daemon ctx failed")
		}
	}()

	if proc != nil { // i am client, daemon created, proc is handle to it
		return proc.Pid, nil
	}

	// i am daemon here
	return 0, DaemonMain(ctx)
}

func EnsureRunning(ctx context.Context) error {
	_, errSearch := _daemonCtx.Search()
	if errSearch == nil {
		return nil
	}

	_, errRestart := startDaemon(ctx)
	if errRestart != nil {
		return errRestart
	}

	tries := 5
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if pmDaemon, errClient := client.New(); errClient == nil {
			if _, errPing := pmDaemon.HealthCheck(ctx); errPing == nil {
				return nil
			}
		}

		<-ticker.C

		tries--
		if tries == 0 {
			return xerr.NewM("daemon didn't started in time")
		}
	}
}

// Restart daemon and get it's pid.
func Restart(ctx context.Context) (int, error) {
	if !daemon.AmIDaemon() {
		if errKill := Kill(); errKill != nil {
			return 0, xerr.NewWM(errKill, "kill daemon to restart")
		}
	}

	return startDaemon(ctx)
}

func readPmConfig() (core.Config, error) {
	config, errRead := core.ReadConfig()
	if errRead != nil {
		if errRead != core.ErrConfigNotExists {
			return fun.Zero[core.Config](), xerr.NewWM(errRead, "read config for migrate")
		}

		log.Info().Msg("writing initial config...")

		if errWrite := core.WriteConfig(core.DefaultConfig); errWrite != nil {
			return fun.Zero[core.Config](), xerr.NewWM(errWrite, "write initial config")
		}

		return core.DefaultConfig, nil
	}

	return config, nil
}

func migrateConfig(config core.Config) error {
	if config.Version == core.Version {
		return nil
	}

	config.Version = core.Version
	if errWrite := core.WriteConfig(config); errWrite != nil {
		return xerr.NewWM(errWrite, "write config for migrate", xerr.Fields{"version": core.Version})
	}

	return nil
}

func unaryLoggerInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	response, err := handler(ctx, req)

	log.Info().
		Str("@request.type", reflect.TypeOf(req).Elem().Name()).
		Any("request", req).
		Str("@response.type", reflect.TypeOf(response).Elem().Name()).
		Any("response", response).
		Err(err).
		Msg(info.FullMethod)

	return response, err
}

func streamLoggerInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	err := handler(srv, ss)

	// log method and resulting error if any
	log.Err(err).Msg(info.FullMethod)

	return err
}

func DaemonMain(ctx context.Context) error {
	log.Logger = zerolog.New(os.Stderr).With().
		Timestamp().
		Caller().
		Logger()

	return fx.New(newApp()).Start(ctx)
}
