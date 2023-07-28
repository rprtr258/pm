package daemon

import (
	"context"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/exp/slog"

	"github.com/rprtr258/pm/internal/core"
)

type watcherEntry struct {
	rootDir string
	pattern *regexp.Regexp
	fn      func() error
}

type watcher struct {
	watcher     *fsnotify.Watcher
	watchplaces map[core.ProcID]watcherEntry
}

func (w watcher) start(ctx context.Context) {
	for _, wp := range w.watchplaces {
		if err := filepath.Walk(wp.rootDir, w.walker); err != nil {
			slog.Error(
				"initial walk failed",
				slog.String("rootDir", wp.rootDir),
				slog.Any("err", err),
			)
		}
	}

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

				if err := wp.fn(); err != nil {
					slog.Error(
						"call watcher function failed",
						slog.Uint64("procID", uint64(procID)),
						slog.String("event", e.String()),
						slog.Any("err", err),
					)
				}
			}

			if e.Op&fsnotify.Create != 0 && stat.IsDir() {
				if err := filepath.Walk(e.Name, w.walker); err != nil {
					slog.Error(
						"walk for created dir failed",
						slog.String("dirname", e.Name),
						slog.Any("err", err),
					)
				}
			}
		case err := <-w.watcher.Errors:
			slog.Error("fsnotify error", slog.Any("err", err))
		}
	}
}

func (w watcher) walker(path string, f os.FileInfo, err error) error {
	if err != nil || !f.IsDir() {
		return nil //nolint:nilerr // skip if not dir
	}

	if err := w.watcher.Add(path); err != nil {
		slog.Error(
			"watch new path",
			slog.String("path", path),
			slog.Any("err", err),
		)
	}

	return nil
}
