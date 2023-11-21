package watcher

import (
	"context"
	"io/fs"
	"os"
	"regexp"
	"time"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
)

type WatcherEntry struct {
	RootDir     string
	Pattern     *regexp.Regexp
	LastModTime time.Time
}

type Watcher struct {
	Watchplaces map[core.PMID]WatcherEntry
}

func New() Watcher {
	return Watcher{
		Watchplaces: make(map[core.PMID]WatcherEntry),
	}
}

func (w Watcher) Add(procID core.PMID, dir, pattern string) error {
	log.Info().
		Stringer("pmid", procID).
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

	w.Watchplaces[procID] = WatcherEntry{
		RootDir: dir,
		Pattern: re,
	}

	return nil
}

func (w Watcher) Remove(procID core.PMID) {
	log.Info().
		Stringer("pmid", procID).
		Msg("removing watch dir")

	delete(w.Watchplaces, procID)
}

func (w Watcher) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// TODO: rewrite
	for {
		select {
		case <-ctx.Done():
			return
		// case event := <-w.statusCh:
		// 	switch e := event.Data.(type) {
		// 	case eventbus.DataProcStarted:
		// 		if _, ok := w.Watchplaces[e.Proc.ID]; !ok && e.EmitReason&^eventbus.EmitReasonByWatcher != 0 {
		// 			if watch, ok := e.Proc.Watch.Unpack(); ok {
		// 				if err := w.Add(e.Proc.ID, e.Proc.Cwd, watch); err != nil {
		// 					log.Error().
		// 						Err(err).
		// 						Stringer("pmid", e.Proc.ID).
		// 						Str("watch", watch).
		// 						Str("cwd", e.Proc.Cwd).
		// 						Msg("add watch failed")
		// 				}
		// 			}
		// 		}
		// 	case eventbus.DataProcStopped:
		// 		if _, ok := w.Watchplaces[e.ProcID]; !ok && e.EmitReason&^eventbus.EmitReasonByWatcher != 0 {
		// 			w.Remove(e.ProcID)
		// 		}
		// 	}
		case now := <-ticker.C:
			// TODO: make concurrent
			for /*id*/ _, wp := range w.Watchplaces {
				updated := false
				fs.WalkDir(os.DirFS(wp.RootDir), "/", func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return fs.SkipAll
					}

					if d.IsDir() {
						return nil
					}

					info, errInfo := d.Info()
					if errInfo != nil {
						return errInfo
					}

					if !wp.Pattern.MatchString(info.Name()) || info.ModTime().Before(wp.LastModTime) {
						return nil
					}

					updated = true
					return fs.SkipAll
				})
				// log.Debug().
				// 	Stringer("pmid", id).
				// 	Str("path", e.Path()).
				// 	Str("root", wp.RootDir).
				// 	Str("pattern", wp.Pattern.String()).
				// 	Str("event", e.Event().String()).
				// 	Msg("watcher got event")

				// TODO: merge into restart request?
				if updated {
					wp.LastModTime = now
					// w.ebus.Publish(ctx,
					// 	eventbus.NewPublishProcStopRequest(id, eventbus.EmitReasonByWatcher),
					// 	eventbus.NewPublishProcStartRequest(id, eventbus.EmitReasonByWatcher),
					// )
				}
			}
		}
	}
}
