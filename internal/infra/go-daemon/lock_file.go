package daemon

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"

	"github.com/rprtr258/xerr"
)

// LockFile wraps *os.File and provide functions for locking of files.
type LockFile struct {
	*os.File
}

// NewLockFile returns a new LockFile with the given File.
func NewLockFile(file *os.File) *LockFile {
	return &LockFile{file}
}

// CreatePidFile opens the named file, applies exclusive lock and writes
// current process id to file.
func CreatePidFile(name string, perm os.FileMode) (*LockFile, error) {
	lock, errOpen := OpenLockFile(name, perm)
	if errOpen != nil {
		return nil, errOpen
	}

	if errLock := lock.Lock(); errLock != nil {
		if errRm := lock.Remove(); errRm != nil {
			return nil, xerr.Combine(errRm, errLock)
		}

		return nil, errLock
	}

	if errWrite := lock.WritePid(); errWrite != nil {
		if errRm := lock.Remove(); errRm != nil {
			return nil, xerr.Combine(errRm, errWrite)
		}

		return nil, errWrite
	}

	return lock, nil
}

// OpenLockFile opens the named file with flags os.O_RDWR|os.O_CREATE and specified perm.
// If successful, function returns LockFile for opened file.
func OpenLockFile(name string, perm os.FileMode) (*LockFile, error) {
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, perm)
	if err != nil {
		return nil, xerr.NewWM(err, "open lock file")
	}

	return &LockFile{file}, nil
}

// Lock apply exclusive lock on an open file. If file already locked, returns error.
func (file *LockFile) Lock() error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK { //nolint:errorlint,goerr113 // check exactly
			return xerr.NewM("file locking would block", xerr.Fields{"filename": file.Name()})
		}

		return xerr.NewWM(err, "lock file", xerr.Fields{"filename": file.Name()})
	}

	return nil
}

// Unlock remove exclusive lock on an open file.
func (file *LockFile) Unlock() error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		if xerr.Is(err, syscall.EWOULDBLOCK) {
			return xerr.NewM("file unlocking would block", xerr.Fields{"filename": file.Name()})
		}

		return xerr.NewWM(err, "unlock file", xerr.Fields{"filename": file.Name()})
	}

	return nil
}

type PidFileNotFoundError struct {
	pidfile string
	exists  bool
}

func (e *PidFileNotFoundError) Error() string {
	if e.exists {
		return fmt.Sprintf("pidfile %s not found", e.pidfile)
	}

	return fmt.Sprintf("pidfile %s is empty or has invalid content", e.pidfile)
}

// ReadPidFile reads process id from file with give name and returns pid.
func ReadPidFile(name string) (int, error) {
	file, err := os.OpenFile(name, os.O_RDONLY, 0o640)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, &PidFileNotFoundError{name, false}
		}

		return 0, xerr.NewWM(err, "open file", xerr.Fields{"pidfile": name})
	}
	defer file.Close()

	lock := &LockFile{file}
	pid, err := lock.ReadPid()
	if err != nil {
		if xerr.Is(err, io.EOF) {
			return 0, &PidFileNotFoundError{name, true}
		}

		return 0, xerr.NewWM(err, "read pid from pidfile", xerr.Fields{"pidfile": name})
	}

	return pid, nil
}

// WritePid writes current process id to an open file.
func (file *LockFile) WritePid() error {
	if _, errSeek := file.Seek(0, io.SeekStart); errSeek != nil {
		return xerr.NewWM(errSeek, "seek 0", xerr.Fields{"filename": file.Name()})
	}

	fileLen, errWrite := fmt.Fprint(file, os.Getpid())
	if errWrite != nil {
		return xerr.NewWM(errWrite, "write pid to file", xerr.Fields{"filename": file.Name()})
	}

	if errTruncate := file.Truncate(int64(fileLen)); errTruncate != nil {
		return xerr.NewWM(errTruncate, "truncate file", xerr.Fields{"filename": file.Name(), "length": fileLen})
	}

	if errSync := file.Sync(); errSync != nil {
		return xerr.NewWM(errSync, "sync file", xerr.Fields{"filename": file.Name()})
	}

	return nil
}

// ReadPid reads process id from file and returns pid.
// If unable read from a file, returns error.
func (file *LockFile) ReadPid() (int, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, xerr.NewWM(err, "seek 0", xerr.Fields{"filename": file.Name()})
	}

	var pid int
	if _, err := fmt.Fscan(file, &pid); err != nil {
		return 0, xerr.NewWM(err, "scan pid", xerr.Fields{"filename": file.Name()})
	}

	return pid, nil
}

// Remove removes lock, closes and removes an open file.
func (file *LockFile) Remove() error {
	defer file.Close()

	if err := file.Unlock(); err != nil {
		return err
	}

	if errRm := os.Remove(file.Name()); errRm != nil {
		return xerr.NewWM(errRm, "remove file", xerr.Fields{"filename": file.Name()})
	}

	return nil
}
