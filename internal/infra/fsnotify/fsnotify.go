// Package fsnotify is a light wrapper around github.com/fsnotify/fsnotify that
// allows for recursively watching directories, and provides a simple wrapper
// for batching events.
package fsnotify

import (
	"io"

	"github.com/fsnotify/fsnotify"
)

// Provide access to the underlying fsnotify ops.
const (
	Chmod  = fsnotify.Chmod
	Create = fsnotify.Create
	Remove = fsnotify.Remove
	Rename = fsnotify.Rename
	Write  = fsnotify.Write
)

type Event = fsnotify.Event

// Option is used to further configure a Watcher or BatchedWatcher.
type Option option

type option func(p *options)

type options struct {
	debug io.Writer
}

// Debug configures debug-level logging via the io.Writer w.
func Debug(w io.Writer) Option {
	return func(o *options) {
		o.debug = w
	}
}
