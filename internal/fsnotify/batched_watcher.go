package fsnotify

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// BatchedRecursiveWatcher is an instance of a recursive watcher that batches
// events. Create a new BatchedRecursiveWatcher via NewBatchedRecursiveWatcher.
type BatchedRecursiveWatcher struct {
	// w is the underlying Watcher used to batch events
	w *RecursiveWatcher

	// Events is the channel over which batches of Events are sent.
	Events chan []fsnotify.Event

	// Errors is the channel over which any Errors are reported
	Errors chan error

	// ticker is set if the run loop is in the process of batching events.
	ticker *time.Ticker

	// batchWindow is the maximum length of a batch window.
	batchWindow time.Duration

	// buffer is the buffer into which we batch events
	buffer []fsnotify.Event

	// inGitOperation is set when a git operation is in progress.
	inGitOperation bool

	// doneClose indicates that we are done handling the close from the
	// underlying fsnotify
	doneClose chan struct{}

	// send is set to non-nil when a batch of events is ready to be sent on.
	// When send != nil, no timer ticks will be set. Instead, if something needs
	// to queue in the time we are waiting to send on a batch it will simply add
	// to the back of the queue. This allows for slow consumers (which we log).
	send chan []fsnotify.Event
}

// NewBatchedRecursiveWatcher creates a new batched watcher. A
// BatchedRecursiveWatcher's Events are batched into windows of at most
// batchWindow duration. Furthermore, if gittoplevel is supplied, events
// received during git operations are batched until the end of the git
// operation. This can mean that batch time windows can be less than
// batchWindow when a git operation starts before a time window expires. It can
// also mean that a batch captures events over a time period greater than
// batchWindow, when a git operation exceeds this duration.
func NewBatchedRecursiveWatcher(
	dir, gittoplevel string,
	batchWindow time.Duration,
) (*BatchedRecursiveWatcher, error) {
	w, err := newRecursiveWatcher(dir, gittoplevel)
	if err != nil {
		return nil, err
	}

	res := &BatchedRecursiveWatcher{ //nolint:exhaustruct // not needed
		w:           w,
		Events:      make(chan []fsnotify.Event),
		Errors:      w.Errors,
		batchWindow: batchWindow,
		doneClose:   make(chan struct{}),
	}
	go res.runEventLoop()
	return res, nil
}

// Close shuts down the watcher, by removing all watches and closing the Events channel.
func (bw *BatchedRecursiveWatcher) Close() error {
	if err := bw.w.Close(); err != nil {
		return fmt.Errorf("failed to shutdown underlying Watcher: %w", err)
	}
	<-bw.doneClose
	return nil
}

// runEventLoop is the main event loop of a BatchedWatcher.
// It is responsible for batching events from the underlying Watcher.
//
//nolint:gocognit // Sadge
func (bw *BatchedRecursiveWatcher) runEventLoop() {
	gitDir := bw.w.gitDir
	lockFile := bw.w.gitLockFile
LOOP:
	for {
		var tick <-chan time.Time
		if bw.ticker != nil {
			tick = bw.ticker.C
		}
		select {
		case ev, ok := <-bw.w.Events:
			log.Debug().Stringer("event", ev).Bool("ok", ok).Msg("batched recursive watcher event")
			if !ok {
				// Pass on the close
				close(bw.Events)
				break LOOP
			}
			var startOfGitOp, endOfGitOp bool
			switch {
			case ev.Name == lockFile:
				startOfGitOp = !bw.inGitOperation && ev.Op == fsnotify.Create
				endOfGitOp = bw.inGitOperation && (ev.Op == fsnotify.Rename || ev.Op == fsnotify.Remove)
				if startOfGitOp {
					log.Debug().Str("operation", "start").Msg("git")
					bw.inGitOperation = true
				}
				if endOfGitOp {
					log.Debug().Str("operation", "end").Msg("git")
					bw.inGitOperation = false
				}
				// We don't care about other events on the lock file during a git
				// op
				if !startOfGitOp && !endOfGitOp {
					continue
				}
				// Proceed like a normal event at this point
			case ev.Name == gitDir || filepath.Dir(ev.Name) == gitDir:
				continue
			}

			// Only create a ticker if we are not on the edge of a git operation
			// and not waiting to send. Because if we are on the edge, then we
			// flush below so creating a ticker would be pointless.
			if bw.send == nil && !startOfGitOp && !endOfGitOp && bw.ticker == nil {
				bw.ticker = time.NewTicker(bw.batchWindow)
			}

			if bw.send != nil {
				// We have a slow consumer
				log.Debug().
					Str("path", ev.Name).
					Stringer("op", ev.Op).
					Msg("slow consumer, adding to existing batch")
			}
			bw.buffer = append(bw.buffer, ev)

			// Again we only flush if we are not already waiting to send.
			// There is no point flushing at the start of a git op, because it's
			// possible/likely that there will be further changes.
			// Triggering an event on this edge therefore increases the likelihood
			// of there being a race in the code that reacts to a change.
			// So only trigger at the end of the git op, by
			// which point we might well have gathered more events.
			if bw.send == nil && endOfGitOp {
				bw.flush()
			}
		case <-tick:
			if len(bw.buffer) == 0 {
				panic("bw.buffer invariant violated: got a timer tick but our buffer was empty")
			}
			bw.flush()
		case bw.send <- bw.buffer:
			bw.send = nil
			bw.buffer = nil
		}
	}
	close(bw.doneClose)
}

// flush triggers a send of any batched events and ensures the ticker is reset if it is active.
func (bw *BatchedRecursiveWatcher) flush() {
	// bw.ticker would be nil if we are eagerly flushing at the start of a git operation.
	if bw.ticker != nil {
		bw.ticker.Stop()
	}
	bw.ticker = nil
	if len(bw.buffer) == 0 {
		return
	}
	bw.send = bw.Events
}
