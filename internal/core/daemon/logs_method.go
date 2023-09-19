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
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
)

type fileSize int64

const (
	_byte     fileSize = 1
	_kibibyte fileSize = 1024 * _byte

	_procLogsBufferSize = 128 * _kibibyte
	_defaultLogsOffset  = 1000 * _byte
)

func streamFile(
	ctx context.Context,
	logLinesCh chan *pb.LogLine,
	procID core.ProcID,
	logFile string,
	logLineType pb.LogLine_Type,
	wgGlobal *sync.WaitGroup,
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
	go func() {
		if err := tailer.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
			select {
			case <-ctx.Done():
				wgGlobal.Done()
				return nil
			case logLinesCh <- &pb.LogLine{
				Line: string(l.Data),
				Time: timestamppb.Now(),
				Type: logLineType,
			}:
				return nil
			}
		}); err != nil {
			log.Error().
				Uint64("procID", procID).
				Str("file", logFile).
				Err(err).
				Msg("failed to tail log")
			// TODO: somehow call wg.Done() once with parent call
		}
	}()

	return nil
}

func (srv *daemonServer) Logs(req *pb.ProcID, stream pb.Daemon_LogsServer) error {
	id := req.GetId()

	// can't get incoming query in interceptor, so logging here also
	log.Info().Uint64("proc_id", id).Msg("Logs method called")

	procs := srv.db.GetProcs(core.WithIDs(id))
	done := make(chan struct{})

	proc, ok := procs[id]
	if !ok {
		return xerr.NewM("proc not found", xerr.Fields{"proc_id": id})
	}

	var wgGlobal sync.WaitGroup
	wgGlobal.Add(2)
	go func(id core.ProcID) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wgLocal sync.WaitGroup

		logLinesCh := make(chan *pb.LogLine)

		wgLocal.Add(1)
		if err := streamFile(
			ctx,
			logLinesCh,
			proc.ID,
			proc.StdoutFile,
			pb.LogLine_TYPE_STDOUT,
			&wgGlobal,
		); err != nil {
			log.Error().
				Uint64("procID", id).
				Str("file", proc.StdoutFile).
				Err(err).
				Msg("failed to stream stdout log file")
		}

		wgLocal.Add(1)
		if err := streamFile(
			ctx,
			logLinesCh,
			proc.ID,
			proc.StderrFile,
			pb.LogLine_TYPE_STDERR,
			&wgGlobal,
		); err != nil {
			log.Error().
				Uint64("procID", id).
				Str("file", proc.StderrFile).
				Err(err).
				Msg("failed to stream stderr log file")
		}

		go func() {
			wgLocal.Wait()
			close(logLinesCh)
		}()

		for {
			select {
			case <-done:
				return
			case line, ok := <-logLinesCh:
				if !ok {
					return
				}

				// TODO: stop on proc death
				if errSend := stream.Send(&pb.ProcsLogs{
					Id:    id,
					Lines: []*pb.LogLine{line}, // TODO: collect lines for some time and send all at once
				}); errSend != nil {
					log.Error().
						Err(errSend).
						Uint64("procID", id).
						Msg("failed to send log lines")
					return
				}
			}
		}
	}(id)

	allStopped := make(chan struct{})
	go func() {
		wgGlobal.Wait()
		close(allStopped)
	}()

	go func() {
		defer close(done)

		select {
		case <-allStopped:
			return
		case <-stream.Context().Done():
			return
		}
	}()

	<-done
	return nil
}
