package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// Mark of daemon process - system environment variable _GO_DAEMON=1
const (
	MARK_NAME  = "_GO_DAEMON"
	MARK_VALUE = "1"
)

// Default file permissions for log and pid files.
const FILE_PERM = os.FileMode(0640)

// WasReborn returns true in child process (daemon) and false in parent process.
func WasReborn() bool {
	return os.Getenv(MARK_NAME) == MARK_VALUE
}

// Reborn runs second copy of current process in the given context.
// function executes separate parts of code in child process and parent process
// and provides demonization of child process. It look similar as the
// fork-daemonization, but goroutine-safe.
// In success returns *os.Process in parent process and nil in child process.
// Otherwise returns error.
func (d *Context) Reborn() (*os.Process, error) {
	if WasReborn() {
		err := d.child()
		if err != nil {
			return nil, fmt.Errorf("reborn child failed: %w", err)
		}
		return nil, nil
	}

	child, err := d.parent()
	if err != nil {
		return nil, fmt.Errorf("reborn parent failed: %w", err)
	}
	return child, nil
}

// Search searches daemons process by given in context pid file name.
// If success returns pointer on daemons os.Process structure,
// else returns error. Returns nil if filename is empty.
func (d *Context) Search() (daemon *os.Process, err error) {
	if len(d.PidFileName) > 0 {
		var pid int
		if pid, err = ReadPidFile(d.PidFileName); err != nil {
			return
		}
		daemon, err = os.FindProcess(pid)
		if err == nil && daemon != nil {
			// Send a test signal to test if this daemon is actually alive or dead
			// An error means it is dead
			if daemon.Signal(syscall.Signal(0)) != nil {
				daemon = nil
			}
		}
	}
	return
}

// Release provides correct pid-file release in daemon.
func (d *Context) Release() error {
	return d.release()
}
