package watcher

import (
	"context"
	"path/filepath"
	"regexp"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/fsnotify"
)

type Entry struct {
	RootDir     string
	Pattern     *regexp.Regexp
	LastModTime time.Time
}

type Watcher struct {
	dir      string
	re       *regexp.Regexp
	watcher  *fsnotify.BatchedRecursiveWatcher
	callback func(context.Context) error
}

func New(dir, pattern string, callback func(context.Context) error) (Watcher, error) {
	watcher, err := fsnotify.NewBatchedRecursiveWatcher(dir, "", time.Second)
	if err != nil {
		return fun.Zero[Watcher](), errors.Wrap(err, "create fsnotify watcher")
	}

	re, errCompilePattern := regexp.Compile(pattern)
	if errCompilePattern != nil {
		return fun.Zero[Watcher](), errors.Wrap(errCompilePattern, "compile pattern")
	}

	return Watcher{
		dir:      dir,
		re:       re,
		watcher:  watcher,
		callback: callback,
	}, nil
}

func (w Watcher) processEventBatch(ctx context.Context, events []fsnotify.Event) {
	triggered := false
	for _, event := range events {
		filename, err := filepath.Rel(w.dir, event.Name)
		if err != nil {
			log.Error().
				Err(err).
				Stringer("event", event).
				Str("dir", w.dir).
				Msg("get relative filename failed")
			continue
		}

		if !w.re.MatchString(filename) {
			continue
		}

		triggered = true
		break
	}

	if triggered {
		if err := w.callback(ctx); err != nil {
			log.Error().
				Err(err).
				Msg("execute callback failed")
		}
	}
}

func (w Watcher) Start(ctx context.Context) {
	defer w.watcher.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-w.watcher.Errors():
			log.Error().
				Err(err).
				Msg("fsnotify error")
			return
		case events := <-w.watcher.Events():
			w.processEventBatch(ctx, events)
		}
	}
}
