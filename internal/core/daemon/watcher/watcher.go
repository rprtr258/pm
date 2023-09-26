package watcher

import (
	"context"
	"regexp"

	"github.com/rjeczalik/notify"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
)

type WatcherEntry struct {
	RootDir string
	Pattern *regexp.Regexp
	ch      chan notify.EventInfo
}

type Watcher struct {
	Watchplaces map[core.ProcID]WatcherEntry
	ebus        *eventbus.EventBus
	statusCh    <-chan eventbus.Event
}

func New(ebus *eventbus.EventBus) Watcher {
	return Watcher{
		Watchplaces: make(map[core.ProcID]WatcherEntry),
		statusCh: ebus.Subscribe(
			"watcher",
			eventbus.KindProcStarted,
			eventbus.KindProcStopped,
		),
		ebus: ebus,
	}
}

func (w Watcher) Add(procID core.ProcID, dir, pattern string) error {
	log.Info().
		Uint64("proc_id", procID).
		Str("dir", dir).
		Str("pattern", pattern).
		Msg("adding watch dir")

	if _, ok := w.Watchplaces[procID]; ok {
		// already added
		return nil
	}

	re, errCompilePattern := regexp.Compile(pattern)
	if errCompilePattern != nil {
		return xerr.NewWM(errCompilePattern, "compile pattern")
	}

	ch := make(chan notify.EventInfo, 1)

	if errWatch := notify.Watch(dir, ch, notify.InCloseWrite); errWatch != nil {
		return xerr.NewWM(errWatch, "add watch dir")
	}

	w.Watchplaces[procID] = WatcherEntry{
		ch:      ch,
		RootDir: dir,
		Pattern: re,
	}

	return nil
}

func (w Watcher) Remove(procID core.ProcID) {
	log.Info().
		Uint64("proc_id", procID).
		Msg("removing watch dir")

	if entry, ok := w.Watchplaces[procID]; ok {
		notify.Stop(entry.ch)
		close(entry.ch) // TODO: reuse channels?
	}
}

func (w Watcher) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-w.statusCh:
			switch e := event.Data.(type) {
			case eventbus.DataProcStarted:
				if _, ok := w.Watchplaces[e.Proc.ID]; !ok && e.EmitReason&^eventbus.EmitReasonByWatcher != 0 {
					if watch, ok := e.Proc.Watch.Unpack(); ok {
						if err := w.Add(e.Proc.ID, e.Proc.Cwd, watch); err != nil {
							log.Error().
								Err(err).
								Uint64("proc_id", e.Proc.ID).
								Str("watch", watch).
								Str("cwd", e.Proc.Cwd).
								Msg("add watch failed")
						}
					}
				}
			case eventbus.DataProcStopped:
				if _, ok := w.Watchplaces[e.ProcID]; !ok && e.EmitReason&^eventbus.EmitReasonByWatcher != 0 {
					w.Remove(e.ProcID)
				}
			}
		default:
			for id, wp := range w.Watchplaces {
				var e notify.EventInfo
				select {
				case e = <-wp.ch:
				default:
					continue
				}
				log.Debug().
					Uint64("proc_id", id).
					Str("path", e.Path()).
					Str("root", wp.RootDir).
					Str("pattern", wp.Pattern.String()).
					Str("event", e.Event().String()).
					Msg("watcher got event")

				if !wp.Pattern.MatchString(e.Path()) {
					continue
				}

				// TODO: merge into restart request?
				w.ebus.Publish(ctx,
					eventbus.NewPublishProcStopRequest(id, eventbus.EmitReasonByWatcher),
					eventbus.NewPublishProcStartRequest(id, eventbus.EmitReasonByWatcher),
				)
			}
		}
	}
}
