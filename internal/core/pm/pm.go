package pm

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/go-faster/tail"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/cli/log"
	"github.com/rprtr258/pm/pkg/client"
)

type App struct {
	client client.Client
	config core.Config
}

func New(pmClient client.Client) (App, error) {
	config, errConfig := core.ReadConfig()
	if errConfig != nil {
		if errConfig == core.ErrConfigNotExists {
			return App{
				client: pmClient,
				config: core.DefaultConfig,
			}, nil
		}

		return fun.Zero[App](), xerr.NewWM(errConfig, "read app config")
	}

	return App{
		client: pmClient,
		config: config,
	}, nil
}

func (app App) CheckDaemon(ctx context.Context) (client.Status, error) {
	status, errHealth := app.client.HealthCheck(ctx)
	if errHealth != nil {
		return fun.Zero[client.Status](), xerr.NewWM(errHealth, "check daemon health")
	}

	return status, nil
}

func (app App) ListByRunConfigs(ctx context.Context, runConfigs []core.RunConfig) (core.Procs, error) {
	list, errList := app.client.List(ctx)
	if errList != nil {
		return nil, xerr.NewWM(errList, "ListByRunConfigs: list procs")
	}

	procNames := fun.FilterMap[string](runConfigs, func(cfg core.RunConfig) fun.Option[string] {
		return cfg.Name
	})

	configList := lo.PickBy(list, func(_ core.ProcID, procData core.Proc) bool {
		return fun.Contains(procNames, procData.Name)
	})

	return configList, nil
}

func (app App) Signal(
	ctx context.Context,
	signal syscall.Signal,
	procIDs ...core.ProcID,
) ([]core.ProcID, error) {
	if len(procIDs) == 0 {
		return []core.ProcID{}, nil
	}

	for _, id := range procIDs {
		if err := app.client.Signal(ctx, signal, id); err != nil {
			return nil, xerr.NewWM(err, "client.stop", xerr.Fields{"proc_id": id})
		}
	}

	return procIDs, nil
}

func (app App) Stop(ctx context.Context, procIDs ...core.ProcID) error {
	for _, id := range procIDs {
		if err := app.client.Stop(ctx, id); err != nil {
			return xerr.NewWM(err, "client.stop", xerr.Fields{"proc_id": id})
		}
	}

	return nil
}

func (app App) Delete(ctx context.Context, procIDs ...core.ProcID) error {
	for _, id := range procIDs {
		if errDelete := app.client.Delete(ctx, id); errDelete != nil {
			return xerr.NewWM(errDelete, "client.delete", xerr.Fields{"proc_id": id})
		}
	}

	return nil
}

func (app App) List(ctx context.Context) (core.Procs, error) {
	list, errList := app.client.List(ctx)
	if errList != nil {
		return nil, xerr.NewWM(errList, "List: list procs")
	}

	return list, nil
}

// Run - create and start processes, returns ids of created processes.
// ids must be handled before handling error, because it tries to run all
// processes and error contains info about all failed processes, not only first.
func (app App) Run(ctx context.Context, config core.RunConfig) (core.ProcID, error) {
	command, errLook := exec.LookPath(config.Command)
	if errLook != nil {
		// if command is relative and failed to look it up, add workdir first
		if filepath.IsLocal(config.Command) {
			config.Command = filepath.Join(config.Cwd, config.Command)
		}

		command, errLook = exec.LookPath(config.Command)
		if errLook != nil {
			return 0, xerr.NewWM(
				errLook,
				"look for executable path",
				xerr.Fields{"executable": config.Command},
			)
		}
	}

	var merr error
	if command == config.Command { // command contains slash and might be relative
		var errAbs error
		command, errAbs = filepath.Abs(command)
		if errAbs != nil {
			xerr.AppendInto(&merr, xerr.NewWM(
				errAbs,
				"abs",
				xerr.Fields{"command": command},
			))
		}
	}

	request := &pb.CreateRequest{
		Command: command,
		Args:    config.Args,
		Name:    config.Name.Ptr(),
		Cwd:     config.Cwd,
		Tags:    config.Tags,
		Env:     config.Env,
		Watch: fun.OptMap(config.Watch, func(r *regexp.Regexp) string {
			return r.String()
		}).Ptr(),
		StdoutFile: config.StdoutFile.Ptr(),
		StderrFile: config.StdoutFile.Ptr(),
	}
	createdProcIDs, errCreate := app.client.Create(ctx, request)
	if errCreate != nil {
		return 0, xerr.NewWM(
			errCreate,
			"server.create",
			xerr.Fields{"process_options": request},
		)
	}

	if errStart := app.client.Start(ctx, createdProcIDs); errStart != nil {
		return createdProcIDs, xerr.NewWM(errStart, "start processes", xerr.Errors{merr})
	}

	return createdProcIDs, merr
}

// Start already created processes
func (app App) Start(ctx context.Context, ids ...core.ProcID) error {
	for _, id := range ids {
		if errStart := app.client.Start(ctx, id); errStart != nil {
			return xerr.NewWM(errStart, "start processes")
		}
	}

	return nil
}

func (app App) Subscribe(ctx context.Context, id core.ProcID) (<-chan core.Proc, error) {
	ch, errLogs := app.client.Subscribe(ctx, id)
	if errLogs != nil {
		return nil, xerr.NewWM(errLogs, "start processes")
	}

	return ch, nil
}

type fileSize int64

const (
	_byte     fileSize = 1
	_kibibyte fileSize = 1024 * _byte

	_procLogsBufferSize = 128 * _kibibyte
	_defaultLogsOffset  = 1000 * _byte
)

type ProcLine struct {
	Line string
	Type core.LogType
	At   time.Time
	Err  error
}

func streamFile(
	ctx context.Context,
	logLinesCh chan ProcLine,
	procID core.ProcID,
	logFile string,
	logLineType core.LogType,
	wg *sync.WaitGroup,
) error {
	stat, errStat := os.Stat(logFile)
	if errStat != nil {
		return xerr.NewWM(errStat, "stat log file")
	}

	tailer := tail.File(logFile, tail.Config{
		Follow:        true,
		BufferSize:    int(_procLogsBufferSize),
		NotifyTimeout: time.Minute,
		Location: &tail.Location{
			Whence: io.SeekEnd,
			Offset: -fun.Min(stat.Size(), int64(_defaultLogsOffset)),
		},
		Logger:  nil,
		Tracker: nil,
	})
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := tailer.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
			select {
			case <-ctx.Done():
				return nil
			case logLinesCh <- ProcLine{
				Line: string(l.Data),
				At:   time.Now(),
				Type: logLineType,
				Err:  nil,
			}:
				return nil
			}
		}); err != nil {
			logLinesCh <- ProcLine{
				Line: "",
				At:   time.Now(),
				Type: logLineType,
				Err: xerr.NewWM(err, "to tail log", xerr.Fields{
					"procID": procID,
					"file":   logFile,
				}),
			}
			return
		}
	}()

	return nil
}

func streamProcLogs(ctx context.Context, proc core.Proc) <-chan ProcLine {
	logLinesCh := make(chan ProcLine)
	go func() {
		var wg sync.WaitGroup
		if err := streamFile(
			ctx,
			logLinesCh,
			proc.ID,
			proc.StdoutFile,
			core.LogTypeStdout,
			&wg,
		); err != nil {
			log.Error().
				// Uint64("procID", proc.ID).
				Str("file", proc.StdoutFile).
				Err(err).
				Msg("failed to stream stdout log file")
		}
		if err := streamFile(
			ctx,
			logLinesCh,
			proc.ID,
			proc.StderrFile,
			core.LogTypeStderr,
			&wg,
		); err != nil {
			log.Error().
				// Uint64("procID", proc.ID).
				Str("file", proc.StderrFile).
				Err(err).
				Msg("failed to stream stderr log file")
		}
		wg.Wait()
		close(logLinesCh)
	}()
	return logLinesCh
}

// Logs - watch for processes logs
func (app App) Logs(ctx context.Context, id core.ProcID) (<-chan core.LogLine, error) {
	procs, err := app.client.List(ctx)
	if err != nil {
		return nil, err
	} else if _, ok := procs[id]; !ok {
		return nil, xerr.NewWM(err, "logs", xerr.Fields{"proc_id": id})
	}
	proc := procs[id]

	ctx, cancel := context.WithCancel(ctx)
	if proc.Status.Status != core.StatusRunning {
		ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
	}

	ch, errLogs := app.client.Subscribe(ctx, id)
	if errLogs != nil {
		return nil, xerr.NewWM(errLogs, "start processes")
	}

	logsCh := streamProcLogs(ctx, proc)

	res := make(chan core.LogLine)
	go func() {
		defer close(res)
		for {
			select {
			case <-ctx.Done():
				return
				// TODO: stop on proc death
			case /*proc :=*/ <-ch:
				// if proc.Status.Status != core.StatusRunning { // TODO: get back
				cancel()
				// }
			case line, ok := <-logsCh:
				if !ok {
					return
				}

				select {
				case <-ctx.Done():
					return
				case res <- core.LogLine{
					ID:   id,
					Name: proc.Name,
					Line: line.Line,
					Type: line.Type,
					At:   line.At,
					Err:  line.Err,
				}:
				}
			}
		}
	}()
	return res, nil
}
