package watcher

import (
	"context"
	"regexp"
	"time"

	"github.com/rjeczalik/notify"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus/queue"
)

type watcherEntry struct {
	rootDir string
	pattern *regexp.Regexp
	ch      chan notify.EventInfo
}

type Watcher struct {
	watchplaces map[core.ProcID]watcherEntry
	dirs        map[string][]core.ProcID // dir -> proc ids using that dir
	ebus        *eventbus.EventBus
	statusQ     *queue.Queue[eventbus.Event]
}

func Module(ctx context.Context, ebus *eventbus.EventBus) {
	Watcher{
		watchplaces: make(map[core.ProcID]watcherEntry),
		dirs:        make(map[string][]core.ProcID),
		statusQ: ebus.Subscribe(
			"watcher",
			eventbus.KindProcStarted,
			eventbus.KindProcStopped,
		),
		ebus: ebus,
	}.Start(ctx)
}

func (w Watcher) Add(procID core.ProcID, dir, pattern string) error {
	log.Info().
		Uint64("proc_id", procID).
		Str("dir", dir).
		Str("pattern", pattern).
		Msg("adding watch dir")

	if _, ok := w.watchplaces[procID]; ok {
		// already added
		return nil
	}

	re, errCompilePattern := regexp.Compile(pattern)
	if errCompilePattern != nil {
		return xerr.NewWM(errCompilePattern, "compile pattern")
	}

	ch := make(chan notify.EventInfo, 1)

	if errWatch := notify.Watch(dir, ch, notify.All, notify.Remove); errWatch != nil {
		return xerr.NewWM(errWatch, "add watch dir")
	}
	defer notify.Stop(ch)

	w.watchplaces[procID] = watcherEntry{
		ch:      ch,
		rootDir: dir,
		pattern: re,
	}

	return nil
}

func (w Watcher) Remove(procID core.ProcID) {
	log.Info().
		Uint64("proc_id", procID).
		Msg("removing watch dir")

	if entry, ok := w.watchplaces[procID]; ok {
		notify.Stop(entry.ch)
		close(entry.ch) // TODO: reuse channels?
	}
}

func (w Watcher) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			event, ok := w.statusQ.Pop()
			if !ok {
				continue
			}

			switch e := event.Data.(type) {
			case eventbus.DataProcStarted:
				if _, ok := w.watchplaces[e.Proc.ID]; !ok && e.EmitReason&^eventbus.EmitReasonByWatcher != 0 {
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
				if _, ok := w.watchplaces[e.ProcID]; !ok && e.EmitReason&^eventbus.EmitReasonByWatcher != 0 {
					w.Remove(e.ProcID)
				}
			}
		default:
			// TODO: unburst, also for logs
			for id, wp := range w.watchplaces {
				var e notify.EventInfo
				select {
				case e = <-wp.ch:
				default:
					continue
				}

				if !wp.pattern.MatchString(e.Path()) {
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
