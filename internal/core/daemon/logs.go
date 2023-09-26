package daemon

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-faster/tail"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
)

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

func (s *Server) streamProcLogs(ctx context.Context, proc core.Proc) <-chan ProcLine {
	ctx, cancel := context.WithCancel(ctx)
	logLinesCh := make(chan ProcLine)
	go func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			ticker := time.NewTicker(100 * time.Millisecond) // TODO: subscribe to db instead ?
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if proc2, ok := s.db.GetProc(proc.ID); !ok || proc2.Status.Status != core.StatusRunning {
						return
					}
				}
			}
		}()
		if err := streamFile(
			ctx,
			logLinesCh,
			proc.ID,
			proc.StdoutFile,
			core.LogTypeStdout,
			&wg,
		); err != nil {
			log.Error().
				Uint64("procID", proc.ID).
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
				Uint64("procID", proc.ID).
				Str("file", proc.StderrFile).
				Err(err).
				Msg("failed to stream stderr log file")
		}
		wg.Wait()
		close(logLinesCh)
	}()
	return logLinesCh
}

func (s *Server) Logs(ctx context.Context, id core.ProcID) (<-chan core.LogLine, error) {
	// can't get incoming query in interceptor, so logging here also
	log.Info().Uint64("proc_id", id).Msg("Logs method called")

	procs := s.db.GetProcs(core.WithIDs(id))

	proc, ok := procs[id]
	if !ok {
		return nil, xerr.NewM("proc not found", xerr.Fields{"proc_id": id})
	}

	ch := make(chan core.LogLine)
	go func() {
		defer close(ch)

		for line := range s.streamProcLogs(ctx, proc) {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// TODO: stop on proc death
			ch <- core.LogLine{
				ID:   id,
				Name: proc.Name,
				At:   line.At,
				Line: line.Line,
				Type: line.Type,
				Err:  line.Err,
			}
		}
	}()
	return ch, nil
}
