package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/rprtr258/xerr"
	"golang.org/x/sys/unix"
)

// Mark of daemon process - system environment variable _GO_DAEMON=1.
const (
	MarkName  = "_GO_DAEMON"
	MarkValue = "1"
)

// Default file permissions for log and pid files.
const FilePerm = os.FileMode(0o640)

// A Context describes daemon context.
// Struct contains only serializable public fields (!!!)
type Context struct {
	// If PidFileName is non-empty, parent process will try to create and lock
	// pid file with given name. Child process writes process id to file.
	PidFileName string

	// If LogFileName is non-empty, parent process will create file with given name
	// and will link to fd 2 (stderr) for child process.
	LogFileName string

	// If WorkDir is non-empty, the child changes into the directory before
	// creating the process.
	WorkDir string
	// If Chroot is non-empty, the child changes root directory
	Chroot string

	abspath string

	pidFile  *LockFile
	logFile  *os.File
	nullFile *os.File

	// Credential holds user and group identities to be assumed by a daemon-process.
	Credential *syscall.Credential

	// If Env is non-nil, it gives the environment variables for the
	// daemon-process in the form returned by os.Environ.
	// If it is nil, the result of os.Environ will be used.
	Env []string
	// If Args is non-nil, it gives the command-line args for the
	// daemon-process. If it is nil, the result of os.Args will be used.
	Args []string

	// Permissions for new pid file.
	PidFilePerm os.FileMode
	// Permissions for new log file.
	LogFilePerm os.FileMode
	// If Umask is non-zero, the daemon-process call Umask() func with given value.
	Umask int
}

// WasReborn returns true in child process (daemon) and false in parent process.
func WasReborn() bool {
	return os.Getenv(MarkName) == MarkValue
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
			return nil, xerr.NewWM(err, "reborn child failed")
		}
		return nil, nil
	}

	child, err := d.parent()
	if err != nil {
		return nil, xerr.NewWM(err, "reborn parent failed")
	}

	return child, nil
}

// Search searches daemons process by given in context pid file name.
// If success returns pointer on daemons os.Process structure,
// else returns error. Returns nil if filename is empty.
func (d *Context) Search() (*os.Process, error) {
	if len(d.PidFileName) == 0 {
		return nil, nil
	}

	pid, err := ReadPidFile(d.PidFileName)
	if err != nil {
		if _, ok := xerr.As[*PidFileNotFoundError](err); ok {
			// TODO: ???
			return nil, nil
		}

		return nil, xerr.NewWM(err, "read pid file", xerr.Fields{"pidFileName": d.PidFileName})
	}

	daemon, err := os.FindProcess(pid)
	if err != nil {
		return nil, xerr.NewWM(
			err, "find process",
			xerr.Fields{"pid": pid})
	}

	if daemon == nil {
		return nil, nil
	}

	// Send a test signal to test if this daemon is actually alive or dead.
	if err := daemon.Signal(syscall.Signal(0)); err != nil {
		// An error means it is dead.
		return nil, nil
	}

	return daemon, nil
}

// Release provides correct pid-file release in daemon.
func (d *Context) Release() error {
	if !initialized || d.pidFile == nil {
		return nil
	}

	return d.pidFile.Remove()
}

func (d *Context) parent() (*os.Process, error) {
	if err := d.prepareEnv(); err != nil {
		return nil, err
	}

	if err := d.openFiles(); err != nil {
		return nil, err
	}
	defer d.closeFiles()

	attr := &os.ProcAttr{
		Dir:   d.WorkDir,
		Env:   d.Env,
		Files: d.files(),
		Sys: &syscall.SysProcAttr{
			// Chroot:     d.Chroot,
			Credential: d.Credential,
			Setsid:     true,
		},
	}

	child, err := os.StartProcess(d.abspath, d.Args, attr)
	if err != nil {
		err = xerr.NewWM(err, "StartProcess")

		if d.pidFile != nil {
			if err2 := d.pidFile.Remove(); err2 != nil {
				return nil, xerr.Combine(err, xerr.NewWM(err2, "pidFile.Remove"))
			}
		}

		return nil, err
	}

	return child, nil
}

func (d *Context) openFiles() error {
	if d.PidFilePerm == 0 {
		d.PidFilePerm = FilePerm
	}
	if d.LogFilePerm == 0 {
		d.LogFilePerm = FilePerm
	}

	var err error
	if d.nullFile, err = os.Open(os.DevNull); err != nil {
		return xerr.NewWM(err, "open /dev/null")
	}

	if len(d.PidFileName) > 0 {
		if d.PidFileName, err = filepath.Abs(d.PidFileName); err != nil {
			return xerr.NewWM(err, "filepath.Abs", xerr.Fields{"pidFilename": d.PidFileName})
		}

		if d.pidFile, err = OpenLockFile(d.PidFileName, d.PidFilePerm); err != nil {
			return xerr.NewWM(err, "OpenLockFile", xerr.Fields{"pidFilename": d.PidFileName})
		}

		if err = d.pidFile.Lock(); err != nil {
			return xerr.NewWM(err, "pidFile.Lock")
		}

		if len(d.Chroot) > 0 {
			// Calculate PID-file absolute path in child's environment
			if d.PidFileName, err = filepath.Rel(d.Chroot, d.PidFileName); err != nil {
				return xerr.NewWM(err, "filepath.Rel", xerr.Fields{
					"basepath": d.Chroot,
					"targpath": d.PidFileName,
				})
			}

			d.PidFileName = "/" + d.PidFileName
		}
	}

	if len(d.LogFileName) > 0 {
		if d.LogFileName == "/dev/stdout" {
			d.logFile = os.Stdout
		} else if d.LogFileName == "/dev/stderr" {
			d.logFile = os.Stderr
		} else if d.logFile, err = os.OpenFile(
			d.LogFileName,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			d.LogFilePerm,
		); err != nil {
			return xerr.NewWM(err, "open log file", xerr.Fields{
				"filename":    d.LogFileName,
				"permissions": d.LogFilePerm,
			})
		}
	}

	return nil
}

func (d *Context) closeFiles() error {
	close := func(file **os.File) {
		if *file != nil {
			(*file).Close()
			*file = nil
		}
	}
	close(&d.logFile)
	close(&d.nullFile)
	close(&d.pidFile.File)

	return nil
}

func (d *Context) prepareEnv() error {
	var err error
	if d.abspath, err = os.Executable(); err != nil {
		return xerr.NewWM(err, "get executable path")
	}

	if len(d.Args) == 0 {
		d.Args = os.Args
	}

	mark := fmt.Sprintf("%s=%s", MarkName, MarkValue)
	if len(d.Env) == 0 {
		d.Env = os.Environ()
	}
	d.Env = append(d.Env, mark)

	return nil
}

func (d *Context) files() []*os.File {
	log := d.nullFile
	if d.logFile != nil {
		log = d.logFile // TODO: just log=d.logFile ?
	}

	files := []*os.File{
		d.nullFile, // d.rpipe,    // (0) stdin
		log,        // (1) stdout
		log,        // (2) stderr
		d.nullFile, // (3) dup on fd 0 after initialization
	}

	if d.pidFile != nil {
		files = append(files, d.pidFile.File) // (4) pid file
	}

	return files
}

var initialized = false

func (d *Context) child() (err error) {
	if initialized {
		return os.ErrInvalid
	}
	initialized = true

	// create PID file after context decoding to know PID file full path.
	if len(d.PidFileName) > 0 {
		d.pidFile = NewLockFile(os.NewFile(4, d.PidFileName))
		if errWritePid := d.pidFile.WritePid(); errWritePid != nil {
			return errWritePid
		}
		defer func() {
			if err != nil {
				err = xerr.Combine(err, d.pidFile.Remove())
			}
		}()
	}

	if errDup := unix.Dup2(3, 0); errDup != nil {
		return xerr.NewWM(errDup, "Dup2")
	}

	if d.Umask != 0 {
		syscall.Umask(d.Umask)
	}
	if len(d.Chroot) > 0 {
		if errChroot := syscall.Chroot(d.Chroot); errChroot != nil {
			return
		}
	}

	return nil
}
