package fsnotify_test

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/rprtr258/pm/internal/infra/fsnotify"
)

var fDebug = flag.Bool("debug", false, "debug logging for tests")

const (
	setupContextKey     = "setupContextKey"
	rootdirFilename     = ".rootdir"
	gittoplevelFilename = ".gittoplevel"
)

func TestScripts(t *testing.T) {
	t.Parallel()
	testscript.Run(t, testscript.Params{ //nolint:exhaustruct // not needed
		UpdateScripts: os.Getenv("CUE_UPDATE") != "",
		Dir:           "testdata",
		Setup:         setup,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"touch": touchCmd,
			"sleep": sleepCmd,
			"log":   logCmd,
		},
	})
}

func setup(e *testscript.Env) (err error) {
	defer func() {
		r := recover()
		switch r := r.(type) {
		case nil:
		case cmdError:
			err = r
		default:
			panic(r)
		}
	}()

	// Establish $HOME for a clean git configuration
	homeDir := filepath.Join(e.Cd, ".home")
	if err := os.Mkdir(homeDir, 0o777); err != nil {
		return fmt.Errorf("failed to create HOME at %s: %w", homeDir, err)
	}
	e.Setenv("HOME", homeDir)

	s := &setupCtx{ //nolint:exhaustruct // not needed
		Env: e,
		log: &watcherLog{b: bytes.NewBuffer(nil), mu: sync.Mutex{}},
	}
	e.Values[setupContextKey] = s

	if gittoplevel, rootdir, err := findSpecialFiles(e.Cd); err != nil {
		return err
	} else {
		s.rootdir = rootdir
		s.gittoplevel = gittoplevel
	}

	// Run git setup if we have a configured gittoplevel
	if s.gittoplevel != "" {
		rungit(s.Vars, s.gittoplevel)
	}

	// If there is a .batched file in e.Cd, we want a BatchedWatcher. If it has
	// non-empty contents, they should parse to a time.Duration
	var h special
	var herr error
	batchedFn := filepath.Join(e.Cd, ".batched")
	if f, err := os.Open(batchedFn); err == nil {
		bytes, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", batchedFn, err)
		}
		dur := strings.TrimSpace(string(bytes))
		d, err := time.ParseDuration(dur)
		if err != nil {
			return fmt.Errorf("failed to parse time.Duration from contents of %s: %w", batchedFn, err)
		}
		h, herr = batchedWatcher(s, d)
	} else {
		h, herr = watcher(s)
	}
	s.handler = h
	return herr
}

type setupCtx struct {
	*testscript.Env
	handler     special
	log         *watcherLog
	rootdir     string
	gittoplevel string
}

// findSpecialFiles walks the directory rooted at s.e.Cd to find special files
// that indicate where our watcher should be established, but also where the
// git directory is rooted.
func findSpecialFiles(currentDir string) (gittoplevel, rootdir string, _ error) {
	var _rootdir, _gittoplevel *string
	toWalk := []string{currentDir}
Walk:
	for len(toWalk) > 0 {
		var dir string
		dir, toWalk = toWalk[0], toWalk[1:]
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			return "", "", fmt.Errorf("failed in search for %s and %s", rootdirFilename, gittoplevelFilename)
		}
		for _, dirEntry := range dirEntries {
			if dirEntry.IsDir() {
				toWalk = append(toWalk, filepath.Join(dir, dirEntry.Name()))
				continue
			}
			if !dirEntry.Type().IsRegular() {
				continue
			}
			if _rootdir == nil && dirEntry.Name() == rootdirFilename {
				_rootdir = &dir
			}
			if _gittoplevel == nil && dirEntry.Name() == gittoplevelFilename {
				_gittoplevel = &dir
			}
			if _rootdir != nil && _gittoplevel != nil {
				break Walk
			}
		}
	}
	if _rootdir == nil {
		return "", "", fmt.Errorf("failed to find special %s file", rootdirFilename)
	}
	rootdir = *_rootdir
	if _gittoplevel != nil {
		gittoplevel = *_gittoplevel
	}
	return
}

var debugOpt = func() fsnotify.Option {
	if *fDebug {
		return fsnotify.Debug(os.Stderr)
	}
	return nil
}()

func watcher(s *setupCtx) (special, error) {
	w, err := fsnotify.NewRecursiveWatcher(s.rootdir, debugOpt)
	if err != nil {
		return special{}, fmt.Errorf("failed to create a Watcher: %w", err)
	}
	s.Env.Defer(func() {
		w.Close()
	})
	bwh := newBatchedWatcherHandler(s, w, handleEvent)
	s.handler = bwh.Special()
	go bwh.run()
	return s.handler, nil
}

func batchedWatcher(s *setupCtx, d time.Duration) (special, error) {
	bw, err := fsnotify.NewBatchedRecursiveWatcher(s.rootdir, s.gittoplevel, d, debugOpt)
	if err != nil {
		return special{}, fmt.Errorf("failed to create a Watcher: %w", err)
	}
	s.Defer(func() {
		bw.Close()
	})
	bwh := newBatchedWatcherHandler(s, bw, handleSliceEvent)
	go bwh.run()
	return bwh.Special(), nil
}

type special struct {
	Watch chan string
	Wait  chan struct{}
}

func newBatchedWatcherHandler[T any](
	s *setupCtx,
	w fsnotify.Watcher[T],
	handler func(*batchedWatcherHandler[T], string, T) string,
) *batchedWatcherHandler[T] {
	return &batchedWatcherHandler[T]{
		s:       s,
		w:       w,
		watchCh: make(chan string),
		waitCh:  make(chan struct{}),
		handler: handler,
	}
}

type batchedWatcherHandler[T any] struct {
	s       *setupCtx
	w       fsnotify.Watcher[T]
	watchCh chan string
	waitCh  chan struct{}
	handler func(*batchedWatcherHandler[T], string, T) string
}

func (b *batchedWatcherHandler[T]) Special() special {
	return special{b.watchCh, b.waitCh}
}

func (b *batchedWatcherHandler[T]) run() {
	var specialFile string
	for {
		select {
		case f := <-b.watchCh:
			if specialFile != "" {
				panic(fmt.Errorf("specialFile already set to %q; tried to set to %q", specialFile, f))
			}
			specialFile = f
		case evs, ok := <-b.w.Events():
			if !ok {
				// Events have been stopped
				return
			}
			specialFile = b.handler(b, specialFile, evs)
		case err := <-b.w.Errors():
			b.s.log.logf("error: %v", err)
		}
	}
}

func handleEvent(b *batchedWatcherHandler[fsnotify.Event], specialFile string, ev fsnotify.Event) string {
	// Make ev.Name relative for logging
	rel, err := filepath.Rel(b.s.rootdir, ev.Name)
	if err != nil {
		b.s.log.logf("error: failed to derive %q relative to %q: %v", ev.Name, b.s.rootdir, err)
	} else {
		b.s.log.logf("name: %s, op: %v\n", rel, ev.Op)
	}
	if ev.Name == specialFile {
		b.waitCh <- struct{}{}
		return ""
	}
	return specialFile
}

func handleSliceEvent(b *batchedWatcherHandler[[]fsnotify.Event], specialFile string, evs []fsnotify.Event) string {
	var sb strings.Builder
	var sawSpecial bool
	sb.WriteString("events [\n")
	for _, ev := range evs {
		if ev.Name == specialFile {
			sawSpecial = true
		}
		// Make ev.Name relative for logging
		rel, err := filepath.Rel(b.s.rootdir, ev.Name)
		if err != nil {
			b.s.log.logf("error: failed to derive %q relative to %q: %v", ev.Name, b.s.rootdir, err)
		} else {
			sb.WriteString(fmt.Sprintf("  name: %s, op: %v\n", rel, ev.Op))
		}
	}
	sb.WriteString("]\n")
	b.s.log.logf(sb.String())
	if sawSpecial {
		b.waitCh <- struct{}{}
		return ""
	}
	return specialFile
}

func rungit(vars []string, gittoplevel string) {
	run(vars, gittoplevel, "git", "init")
	run(vars, gittoplevel, "git", "config", "user.name", "blah")
	run(vars, gittoplevel, "git", "config", "user.email", "blah@blah.com")
	run(vars, gittoplevel, "git", "add", "-A")
	run(vars, gittoplevel, "git", "commit", "-m", "initial commit")
}

func run(vars []string, dir, cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	c.Dir = dir
	c.Env = vars
	byts, err := c.CombinedOutput()
	if err != nil {
		panic(cmdError{fmt.Errorf("failed to run %v: %w\n%s", c, err, byts)})
	}
}

type cmdError struct {
	error
}

// log dumps the log since the last call to log (or since the beginning of time
// if this is the first such call in a script) to stdout. It optionally takes a
// single argument, a path to the special file to use.
func logCmd(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("log cannot be negated")
	}
	if len(args) > 1 {
		ts.Fatalf("log takes at most one argument")
	}

	sc, ok := ts.Value(setupContextKey).(*setupCtx)
	if !ok {
		ts.Fatalf("failed to find batchedWatcherHandler - are we in a batched watcher test?")
	}

	sf := filepath.Join(sc.rootdir, ".special")
	if len(args) == 1 {
		sf = ts.MkAbs(args[0])
	}
	sf = ts.MkAbs(sf)

	done := make(chan struct{})
	go func() {
		<-sc.handler.Wait
		close(done)
	}()

	// Tell the handler about the special file
	sc.handler.Watch <- sf

	// Now touch the special file
	now := time.Now()
	if err := os.Chtimes(sf, now, now); err != nil {
		ts.Fatalf("failed to touch special file %s: %v", sf, err)
	}

	// Wait till we hear back
	<-done

	// Snapshot the log
	snapshot, err := sc.log.snapshot()
	if err != nil {
		ts.Fatalf("failed to snapshot watcher log: %v", err)
	}
	fmt.Fprintf(ts.Stdout(), "%s", snapshot)
}

// touch takes a list of files to touch, like the unix touch command
func touchCmd(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("touch cannot be negated")
	}
	now := time.Now()
	for _, v := range args {
		v = ts.MkAbs(v)
		if err := os.Chtimes(v, now, now); err != nil {
			ts.Fatalf("failed to touch %s: %v", v, err)
		}
	}
}

// sleep optionally takes a single argument, a time.Duration that can be parsed
// by time.ParseDuration, and sleeps for that length of time. If no duration is
// passed then a sensible default value is used, a default that works in "most"
// situations.
func sleepCmd(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("sleep cannot be negated")
	}
	if len(args) > 1 {
		ts.Fatalf("sleep takes at most one argument")
	}
	var d time.Duration
	var err error
	switch len(args) {
	case 0:
		d = 10 * time.Millisecond
	case 1:
		ds := args[0]
		d, err = time.ParseDuration(ds)
		if err != nil {
			ts.Fatalf("failed to parse %q as a time.Duration: %v", ds, err)
		}
	default:
		panic("should not be here")
	}
	time.Sleep(d)
}

// watcherLog is a mutex-guarded bytes.Buffer. Events are logged to this
// buffer, and periodically the buffer is read by the log testscript builtin.
type watcherLog struct {
	mu sync.Mutex
	b  *bytes.Buffer
}

func (tw *watcherLog) logf(format string, args ...any) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	fmt.Fprintf(tw.b, format, args...)
}

func (tw *watcherLog) snapshot() ([]byte, error) {
	tw.mu.Lock()
	got, err := io.ReadAll(tw.b)
	tw.mu.Unlock()
	return got, err
}
