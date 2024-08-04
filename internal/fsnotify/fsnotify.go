// Package fsnotify is a light wrapper around github.com/fsnotify/fsnotify that
// allows for recursively watching directories, and provides a simple wrapper
// for batching events.
package fsnotify

import "github.com/fsnotify/fsnotify"

type Event = fsnotify.Event
