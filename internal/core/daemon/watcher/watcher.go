package watcher

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
	fn      func(context.Context) error
}

type Watcher struct {
	watcher     *fsnotify.Watcher
	watchplaces map[core.ProcID]watcherEntry
	dirs        map[string][]core.ProcID // dir -> proc ids using that dir
}

func New(watcher *fsnotify.Watcher) Watcher {
	return Watcher{
		watcher:     watcher,
		watchplaces: make(map[core.ProcID]watcherEntry),
		dirs:        make(map[string][]core.ProcID),
	}
}

func (w Watcher) Add(procID core.ProcID, dir string, pattern string, fn func(context.Context) error) {
	slog.Info(
		"adding watch dir",
		slog.Uint64("proc_id", procID),
		slog.String("dir", dir),
		slog.String("pattern", pattern),
	)

	if _, ok := w.watchplaces[procID]; ok {
		return
	}

	re, errCompilePattern := regexp.Compile(pattern)
	if errCompilePattern != nil {
		slog.Error(
			"pattern compilation failed",
			slog.Uint64("proc_id", procID),
			slog.String("pattern", pattern),
			slog.Any("err", errCompilePattern),
		)
		return
	}

	w.watchplaces[procID] = watcherEntry{
		rootDir: dir,
		pattern: re,
		fn:      fn,
	}

	if errWalk := filepath.Walk(dir, w.walker(procID)); errWalk != nil {
		slog.Error(
			"walk failed",
			slog.String("rootDir", dir),
			slog.Any("err", errWalk),
		)
	}
}

func (w Watcher) Remove(procIDs ...core.ProcID) {
	slog.Info(
		"removing watch dirs",
		slog.Any("proc_ids", procIDs),
	)

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
				slog.Error(
					"remove watch on dir failed",
					slog.Any("proc_ids", procIDs),
					slog.String("dir", dir),
					slog.Any("err", errRm),
				)
			}
		}
	}
}

func (w Watcher) Start(ctx context.Context) {
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

				if err := wp.fn(ctx); err != nil {
					slog.Error(
						"call watcher function failed",
						slog.Uint64("procID", procID),
						slog.String("event", e.String()),
						slog.Any("err", err),
					)
				}
			}

			if e.Op&fsnotify.Create != 0 && stat.IsDir() {
				procIDs := w.dirs[filepath.Dir(e.Name)]
				if err := filepath.Walk(e.Name, w.walker(procIDs...)); err != nil {
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
			slog.Error(
				"watch new path",
				slog.String("path", path),
				slog.Any("err", err),
			)
		}

		w.dirs[path] = append(w.dirs[path], procIDs...)

		return nil
	}
}