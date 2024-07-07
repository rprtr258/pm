// Package fsnotify is a light wrapper around github.com/fsnotify/fsnotify that
// allows for recursively watching directories, and provides a simple wrapper
// for batching events.
package fsnotify

import "github.com/fsnotify/fsnotify"

// Provide access to the underlying fsnotify ops.
const (
	Create = fsnotify.Create
	Write  = fsnotify.Write
	Remove = fsnotify.Remove
	Rename = fsnotify.Rename
	Chmod  = fsnotify.Chmod
)

type Event = fsnotify.Event
