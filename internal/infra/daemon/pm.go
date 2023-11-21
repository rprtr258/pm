package daemon

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-faster/tail"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core"
	// "github.com/rprtr258/pm/internal/infra/log"
)

const _envPMID = "PM_PMID"

func (app App) ListByRunConfigs(runConfigs []core.RunConfig) (core.Procs, error) {
	list := app.List()

	procNames := fun.FilterMap[string](runConfigs, func(cfg core.RunConfig) fun.Option[string] {
		return cfg.Name
	})

	configList := lo.PickBy(list, func(_ core.PMID, procData core.Proc) bool {
		return fun.Contains(procNames, procData.Name)
	})

	return configList, nil
}

func procFields(proc core.Proc) string {
	return fmt.Sprintf(
		`Proc[id=%s, command=%q, cwd=%q, name=%q, args=%q, tags=%q, watch=%q, status=%q, stdout_file=%q, stderr_file=%q]`,
		proc.ID,
		proc.Command,
		proc.Cwd,
		proc.Name,
		proc.Args,
		proc.Tags,
		func(opt fun.Option[string]) string {
			if !opt.Valid {
				return "None"
			}

			return fmt.Sprintf("Some(%v)", opt.Value)
		}(proc.Watch),
		proc.Status,
		proc.StdoutFile,
		proc.StderrFile,
		// TODO: uncomment
		// "restart_tries": proc.RestartTries,
		// "restart_delay": proc.RestartDelay,
		// "respawns":     proc.Respawns,
	)
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
	procID core.PMID,
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
func (app App) Logs(ctx context.Context, id core.PMID) (<-chan core.LogLine, error) {
	procs := app.List()
	if _, ok := procs[id]; !ok {
		return nil, xerr.NewM("logs", xerr.Fields{"pmid": id})
	}
	proc := procs[id]

	ctx, cancel := context.WithCancel(ctx)
	if proc.Status.Status != core.StatusRunning {
		ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
	}

	// ch, errLogs := app.Client.Subscribe(ctx, id)
	// if errLogs != nil {
	// 	cancel()
	// 	return nil, xerr.NewWM(errLogs, "start processes")
	// }

	logsCh := streamProcLogs(ctx, proc)

	res := make(chan core.LogLine)
	go func() {
		defer close(res)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			// TODO: stop on proc death
			// case proc := <-ch:
			// 	if proc.Status.Status != core.StatusRunning {
			// 		cancel()
			// 	}
			case line, ok := <-logsCh:
				if !ok {
					return
				}

				select {
				case <-ctx.Done():
					return
				case res <- core.LogLine{
					ProcID:   id,
					ProcName: proc.Name,
					Line:     line.Line,
					Type:     line.Type,
					At:       line.At,
					Err:      line.Err,
				}:
				}
			}
		}
	}()
	return res, nil
}
