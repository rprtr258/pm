package app

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

	procNames := fun.FilterMap[string](func(cfg core.RunConfig) fun.Option[string] {
		return cfg.Name
	}, runConfigs...)

	configList := lo.PickBy(list, func(_ core.PMID, procData core.Proc) bool {
		return fun.Contains(procData.Name, procNames...)
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
				Type: logLineType,
				Err:  nil,
			}:
				return nil
			}
		}); err != nil {
			logLinesCh <- ProcLine{
				Line: "",
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
	proc, ok := app.Get(id)
	if !ok {
		return nil, xerr.NewM("logs", xerr.Fields{"pmid": id})
	}

	ctx, cancel := context.WithCancel(ctx)
	if proc.Status.Status != core.StatusRunning {
		ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	}

	logsCh := streamProcLogs(ctx, proc)

	res := make(chan core.LogLine)
	go func() {
		defer close(res)
		defer cancel()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// TODO: reread database without recreating whole app
				newApp, err := New()
				if err != nil {
					log.Error().Err(err).Msg("failed to check proc status")
				}

				if newApp.List()[id].Status.Status != core.StatusRunning {
					return
				}
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
					Err:      line.Err,
				}:
				}
			}
		}
	}()
	return res, nil
}
