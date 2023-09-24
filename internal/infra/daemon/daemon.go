package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/go-faster/tail"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/runner"
	"github.com/rprtr258/pm/internal/core/daemon/watcher"
	"github.com/rprtr258/pm/internal/core/pm"
	log2 "github.com/rprtr258/pm/internal/infra/cli/log"
	"github.com/rprtr258/pm/internal/infra/cli/log/buffer"
	"github.com/rprtr258/pm/internal/infra/db"
	godaemon "github.com/rprtr258/pm/internal/infra/go-daemon"
	"github.com/rprtr258/pm/pkg/client"
)

var _daemonCtx = &godaemon.Context{
	PidFileName: _filePid,
	PidFilePerm: 0o644, // default pid file permissions, rwxr--r--
	LogFileName: _fileLog,
	LogFilePerm: 0o640, // default log file permissions, rwxr-----
	WorkDir:     "./",
	Umask:       0o27, // don't know
	Args:        []string{"pm", "daemon", "start"},
	Chroot:      "",
	Env:         nil,
	Credential:  nil,
}

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
	for k, v := range status.Status.Envs {
		if !strings.ContainsAny(v, "\n\r\t ") {
			status.Status.Envs[k] = strings.NewReplacer(
				"\n", buffer.String(`\n`, buffer.FgGreen),
				"\r", buffer.String(`\r`, buffer.FgGreen),
				"\t", buffer.String(`\t`, buffer.FgGreen),
				" ", buffer.String(`\x20`, buffer.FgGreen),
			).Replace(v)
		}
	}

	// crop long values
	for k, v := range status.Status.Envs {
		if len(v) <= 100 {
			continue
		}

		status.Status.Envs[k] = v[:50] + buffer.String("...", buffer.FgBlue) + v[len(v)-50:]
	}

	log2.Info().
		Any("Args", status.Status.Args).
		Any("Envs", status.Status.Envs).
		Str("Executable", status.Status.Executable).
		Any("CWD", status.Status.Cwd).
		Any("Groups", status.Status.Groups).
		Any("Page Size:", status.Status.PageSize).
		Any("Hostname", status.Status.Hostname).
		Any("User Cache Dir", status.Status.UserCacheDir).
		Any("User Config Dir", status.Status.UserConfigDir).
		Any("User Home Dir", status.Status.UserHomeDir).
		Any("PID", status.Status.PID).
		Any("PPID", status.Status.PPID).
		Any("UID", status.Status.UID).
		Any("EUID", status.Status.EUID).
		Any("GID", status.Status.GID).
		Any("EGID", status.Status.EGID).
		Any("Watches", status.Watches).
		Msg("Daemon info")

	return nil
}

// Kill daemon. If daemon is already killed, do nothing.
func Kill() error {
	if err := os.Remove(core.SocketRPC); err != nil && !errors.Is(err, os.ErrNotExist) {
		return xerr.NewWM(err, "remove socket file")
	}

	proc, err := _daemonCtx.Search()
	if err != nil {
		if err == godaemon.ErrDaemonNotFound {
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
	case <-time.After(5 * time.Second):
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
	if !godaemon.AmIDaemon() {
		if errKill := Kill(); errKill != nil {
			return 0, xerr.NewWM(errKill, "kill daemon to restart")
		}
	}

	return startDaemon(ctx)
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
	return 0, Main(ctx)
}

func Logs(ctx context.Context, follow bool) error {
	stat, errStat := os.Stat(_fileLog)
	if errStat != nil {
		return xerr.NewWM(errStat, "stat log file", xerr.Fields{"file": _fileLog})
	}

	const _defaultOffset = 10000

	t := tail.File(_fileLog, tail.Config{
		Location: &tail.Location{
			Offset: -min(stat.Size(), _defaultOffset),
			Whence: io.SeekEnd,
		},
		NotifyTimeout: 1 * time.Minute,
		Follow:        follow,
		BufferSize:    64 * 1024, // 64 kb
		Logger:        nil,
		Tracker:       nil,
	})

	if err := t.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
		fmt.Println(string(l.Data))
		return nil
	}); err != nil {
		return xerr.NewWM(err, "tail daemon logs")
	}

	return nil
}

var (
	_filePid = filepath.Join(core.DirHome, "pm.pid")
	_fileLog = filepath.Join(core.DirHome, "pm.log")
	_dirDB   = filepath.Join(core.DirHome, "db") // TODO: remove
)

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

func Main(ctx context.Context) error {
	log.Logger = zerolog.New(os.Stderr).With().
		Timestamp().
		Caller().
		Logger()

	cfg, errCfg := daemon.ReadPMConfig()
	if errCfg != nil {
		return fmt.Errorf("config: %w", errCfg)
	}

	if errMigrate := daemon.MigrateConfig(cfg); errMigrate != nil {
		return fmt.Errorf("migrate: %w", errMigrate)
	}

	dbHandle, errDB := db.New(_dirDB)
	if errDB != nil {
		return fmt.Errorf("db: %w", errDB)
	}

	ebus := eventbus.Module(dbHandle)
	go ebus.Start(ctx)

	watcher := watcher.New(ebus)
	go watcher.Start(ctx)

	go daemon.StartDeathCollector(ctx, ebus, dbHandle)
	go daemon.StartStatuser(ctx, ebus, dbHandle)
	go runner.Start(ctx, ebus, dbHandle)

	srv := daemon.NewServer(ebus, dbHandle, watcher)

	sock, err := net.Listen("unix", core.SocketRPC)
	if err != nil {
		return err
	}
	defer sock.Close()

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryLoggerInterceptor),
		grpc.ChainStreamInterceptor(streamLoggerInterceptor),
	)
	pb.RegisterDaemonServer(s, &daemonServer{
		UnimplementedDaemonServer: pb.UnimplementedDaemonServer{},
		srv:                       srv,
	})
	log.Info().Stringer("socket", sock.Addr()).Msg("daemon started")
	if err := s.Serve(sock); err != nil {
		return err
	}
	defer func() {
		s.GracefulStop()

		if errRm := os.Remove(core.SocketRPC); errRm != nil && !errors.Is(errRm, os.ErrNotExist) {
			log.Error().
				Err(errRm).
				Str("file", _filePid).
				Msg("remove pid file")
		}
	}()

	return nil
}