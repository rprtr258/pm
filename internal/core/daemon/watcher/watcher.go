package watcher

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"

	// fsnotify "github.com/rjeczalik/notify"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus/queue"
)

type watcherEntry struct {
	rootDir string
	pattern *regexp.Regexp
}

type Watcher struct {
	watcher     *fsnotify.Watcher
	watchplaces map[core.ProcID]watcherEntry
	dirs        map[string][]core.ProcID // dir -> proc ids using that dir
	ebus        *eventbus.EventBus
	statusQ     *queue.Queue[eventbus.Event]
}

func Module(ctx context.Context, fsWatcher *fsnotify.Watcher, ebus *eventbus.EventBus) {
	Watcher{
		watcher:     fsWatcher,
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

func (w Watcher) Add(procID core.ProcID, dir, pattern string) {
	log.Info().
		Uint64("proc_id", procID).
		Str("dir", dir).
		Str("pattern", pattern).
		Msg("adding watch dir")

	if _, ok := w.watchplaces[procID]; ok {
		return
	}

	re, errCompilePattern := regexp.Compile(pattern)
	if errCompilePattern != nil {
		log.Error().
			Err(errCompilePattern).
			Uint64("proc_id", procID).
			Str("pattern", pattern).
			Msg("pattern compilation failed")
		return
	}

	w.watchplaces[procID] = watcherEntry{
		rootDir: dir,
		pattern: re,
	}

	if errWalk := filepath.Walk(dir, w.walker(procID)); errWalk != nil {
		log.Error().
			Err(errWalk).
			Str("rootDir", dir).
			Msg("walk failed")
	}
}

func (w Watcher) Remove(procIDs ...core.ProcID) {
	log.Info().
		Uints64("proc_ids", procIDs).
		Msg("removing watch dirs")

	for dir, procs := range w.dirs {
		leftProcIDs := []core.ProcID{}
		for _, procID := range procs {
			take := true
			for _, procID2 := range procIDs {
				if procID == procID2 {
					take = false
					break
				}
			}
			if take {
				leftProcIDs = append(leftProcIDs, procID)
			}
		}

		if len(leftProcIDs) > 0 {
			w.dirs[dir] = leftProcIDs
		} else {
			delete(w.dirs, dir)
			if errRm := w.watcher.Remove(dir); errRm != nil {
				log.Error().
					Err(errRm).
					Uints64("proc_ids", procIDs).
					Str("dir", dir).
					Msg("remove watch on dir failed")
			}
		}
	}
}

func (w Watcher) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		// TODO: unburst, also for logs
		case e := <-w.watcher.Events:
			stat, err := os.Stat(e.Name)
			if err != nil {
				continue
			}

			for procID, wp := range w.watchplaces {
				if !wp.pattern.MatchString(e.Name) {
					continue
				}

				// TODO: merge into restart request?
				w.ebus.Publish(ctx,
					eventbus.NewPublishProcStopRequest(procID, eventbus.EmitReasonByWatcher),
					eventbus.NewPublishProcStartRequest(procID, eventbus.EmitReasonByWatcher),
				)
			}

			if e.Op&fsnotify.Create != 0 && stat.IsDir() {
				procIDs := w.dirs[filepath.Dir(e.Name)]
				if err := filepath.Walk(e.Name, w.walker(procIDs...)); err != nil {
					log.Error().
						Err(err).
						Str("dirname", e.Name).
						Msg("walk for created dir failed")
				}
			}
		case err := <-w.watcher.Errors:
			if err != nil {
				log.Error().Err(err).Msg("fsnotify error")
			} else {
				log.Error().Msg("fsnotify sent nil error")
			}
		case <-ticker.C:
			event, ok := w.statusQ.Pop()
			if !ok {
				continue
			}

			switch e := event.Data.(type) {
			case eventbus.DataProcStarted:
				if _, ok := w.watchplaces[e.Proc.ID]; !ok && e.EmitReason&^eventbus.EmitReasonByWatcher != 0 {
					if watch, ok := e.Proc.Watch.Unpack(); ok {
						w.Add(e.Proc.ID, e.Proc.Cwd, watch)
					}
				}
			case eventbus.DataProcStopped:
				if _, ok := w.watchplaces[e.ProcID]; !ok && e.EmitReason&^eventbus.EmitReasonByWatcher != 0 {
					w.Remove(e.ProcID)
				}
			}
		}
	}
}

func (w Watcher) walker(procIDs ...core.ProcID) filepath.WalkFunc {
	if len(procIDs) == 0 {
		return func(path string, f os.FileInfo, err error) error {
			return filepath.SkipDir
		}
	}

	return func(path string, f os.FileInfo, err error) error {
		if _, ok := w.dirs[path]; ok {
			return filepath.SkipDir
		}

		if err != nil || !f.IsDir() {
			return nil //nolint:nilerr // skip if not dir
		}

		if err := w.watcher.Add(path); err != nil {
			log.Error().
				Err(err).
				Str("path", path).
				Msg("watch new path")
		}

		w.dirs[path] = append(w.dirs[path], procIDs...)

		return nil
	}
}
