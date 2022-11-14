//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || plan9 || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd plan9 solaris

package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"go.uber.org/multierr"
	"golang.org/x/sys/unix"
)

// A Context describes daemon context.
type Context struct {
	// If PidFileName is non-empty, parent process will try to create and lock
	// pid file with given name. Child process writes process id to file.
	PidFileName string
	// Permissions for new pid file.
	PidFilePerm os.FileMode

	// If LogFileName is non-empty, parent process will create file with given name
	// and will link to fd 2 (stderr) for child process.
	LogFileName string
	// Permissions for new log file.
	LogFilePerm os.FileMode

	// If WorkDir is non-empty, the child changes into the directory before
	// creating the process.
	WorkDir string
	// If Chroot is non-empty, the child changes root directory
	Chroot string

	// If Env is non-nil, it gives the environment variables for the
	// daemon-process in the form returned by os.Environ.
	// If it is nil, the result of os.Environ will be used.
	Env []string
	// If Args is non-nil, it gives the command-line args for the
	// daemon-process. If it is nil, the result of os.Args will be used.
	Args []string

	// Credential holds user and group identities to be assumed by a daemon-process.
	Credential *syscall.Credential
	// If Umask is non-zero, the daemon-process call Umask() func with given value.
	Umask int

	// Struct contains only serializable public fields (!!!)
	abspath  string
	pidFile  *LockFile
	logFile  *os.File
	nullFile *os.File

	rpipe, wpipe *os.File
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
			//Chroot:     d.Chroot,
			Credential: d.Credential,
			Setsid:     true,
		},
	}

	child, err := os.StartProcess(d.abspath, d.Args, attr)
	if err != nil {
		err = fmt.Errorf("StartProcess failed: %w", err)

		if d.pidFile != nil {
			err2 := d.pidFile.Remove()
			if err2 != nil {
				err2 = fmt.Errorf("pidFile.Remove failed: %w", err2)
			}

			return nil, multierr.Combine(err, err2)
		}

		return nil, err
	}

	if err := d.rpipe.Close(); err != nil {
		return nil, fmt.Errorf("rpipe.Close failed: %w", err)
	}

	encoder := json.NewEncoder(d.wpipe)
	if err = encoder.Encode(d); err != nil {
		return nil, fmt.Errorf("json encode failed: %w", err)
	}

	if _, err := fmt.Fprint(d.wpipe, "\n\n"); err != nil {
		return nil, fmt.Errorf("print '\n\n' to wpipe failed: %w", err)
	}

	return child, nil
}

func (d *Context) openFiles() error {
	if d.PidFilePerm == 0 {
		d.PidFilePerm = FILE_PERM
	}
	if d.LogFilePerm == 0 {
		d.LogFilePerm = FILE_PERM
	}

	var err error
	if d.nullFile, err = os.Open(os.DevNull); err != nil {
		return fmt.Errorf("open(devNull) failed: %w", err)
	}

	if len(d.PidFileName) > 0 {
		if d.PidFileName, err = filepath.Abs(d.PidFileName); err != nil {
			return fmt.Errorf("abs(pidFile) failed: %w", err)
		}
		if d.pidFile, err = OpenLockFile(d.PidFileName, d.PidFilePerm); err != nil {
			return fmt.Errorf("OpenLockFile(pidFile) failed: %w", err)
		}
		if err = d.pidFile.Lock(); err != nil {
			return fmt.Errorf("pidFile.Lock() failed: %w", err)
		}
		if len(d.Chroot) > 0 {
			// Calculate PID-file absolute path in child's environment
			if d.PidFileName, err = filepath.Rel(d.Chroot, d.PidFileName); err != nil {
				return fmt.Errorf("Rel(%s, pidFile) failed: %w", d.Chroot, err)
			}
			d.PidFileName = "/" + d.PidFileName
		}
	}

	if len(d.LogFileName) > 0 {
		if d.LogFileName == "/dev/stdout" {
			d.logFile = os.Stdout
		} else if d.LogFileName == "/dev/stderr" {
			d.logFile = os.Stderr
		} else if d.logFile, err = os.OpenFile(d.LogFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, d.LogFilePerm); err != nil {
			return fmt.Errorf("OpenFile(logFile, perms=%v) failed: %w", d.LogFilePerm, err)
		}
	}

	d.rpipe, d.wpipe, err = os.Pipe()
	if err != nil {
		return fmt.Errorf("os.Pipe() failed: %w", err)
	}

	return nil
}

func (d *Context) closeFiles() (err error) {
	cl := func(file **os.File) {
		if *file != nil {
			(*file).Close()
			*file = nil
		}
	}
	cl(&d.rpipe)
	cl(&d.wpipe)
	cl(&d.logFile)
	cl(&d.nullFile)
	if d.pidFile != nil {
		d.pidFile.Close()
		d.pidFile = nil
	}
	return
}

func (d *Context) prepareEnv() (err error) {
	if d.abspath, err = os.Executable(); err != nil {
		return
	}

	if len(d.Args) == 0 {
		d.Args = os.Args
	}

	mark := fmt.Sprintf("%s=%s", MARK_NAME, MARK_VALUE)
	if len(d.Env) == 0 {
		d.Env = os.Environ()
	}
	d.Env = append(d.Env, mark)

	return
}

func (d *Context) files() (f []*os.File) {
	log := d.nullFile
	if d.logFile != nil {
		log = d.logFile
	}

	f = []*os.File{
		d.rpipe,    // (0) stdin
		log,        // (1) stdout
		log,        // (2) stderr
		d.nullFile, // (3) dup on fd 0 after initialization
	}

	if d.pidFile != nil {
		f = append(f, d.pidFile.File) // (4) pid file
	}
	return
}

var initialized = false

func (d *Context) child() (err error) {
	if initialized {
		return os.ErrInvalid
	}
	initialized = true

	decoder := json.NewDecoder(os.Stdin)
	if err = decoder.Decode(d); err != nil {
		return
	}

	// create PID file after context decoding to know PID file full path.
	if len(d.PidFileName) > 0 {
		d.pidFile = NewLockFile(os.NewFile(4, d.PidFileName))
		if err = d.pidFile.WritePid(); err != nil {
			return
		}
		defer func() {
			if err != nil {
				d.pidFile.Remove()
			}
		}()
	}

	if err = unix.Dup2(3, 0); err != nil {
		return
	}

	if d.Umask != 0 {
		syscall.Umask(int(d.Umask))
	}
	if len(d.Chroot) > 0 {
		err = syscall.Chroot(d.Chroot)
		if err != nil {
			return
		}
	}

	return
}

func (d *Context) release() error {
	if !initialized || d.pidFile == nil {
		return nil
	}

	return d.pidFile.Remove()
}
