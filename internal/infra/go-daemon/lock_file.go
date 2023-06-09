package daemon

import (
	"fmt"
	"io"
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
func CreatePidFile(name string, perm os.FileMode) (lock *LockFile, err error) {
	if lock, err = OpenLockFile(name, perm); err != nil {
		return
	}
	if err = lock.Lock(); err != nil {
		lock.Remove()
		return
	}
	if err = lock.WritePid(); err != nil {
		lock.Remove()
	}
	return
}

// OpenLockFile opens the named file with flags os.O_RDWR|os.O_CREATE and specified perm.
// If successful, function returns LockFile for opened file.
func OpenLockFile(name string, perm os.FileMode) (lock *LockFile, err error) {
	var file *os.File
	if file, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE, perm); err == nil {
		lock = &LockFile{file}
	}
	return
}

// Lock apply exclusive lock on an open file. If file already locked, returns error.
func (file *LockFile) Lock() error {
	fd := file.Fd()
	err := syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK { //nolint:errorlint,goerr113 // check exactly
			return xerr.NewM("file locking would block", xerr.Fields{"filename": file.Name()})
		}

		return xerr.NewWM(err, "lock file", xerr.Fields{"filename": file.Name()})
	}

	return nil
}

// Unlock remove exclusive lock on an open file.
func (file *LockFile) Unlock() error {
	fd := file.Fd()
	err := syscall.Flock(int(fd), syscall.LOCK_UN)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
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
		if os.IsNotExist(err) {
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
func (file *LockFile) WritePid() (err error) {
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return
	}
	var fileLen int
	if fileLen, err = fmt.Fprint(file, os.Getpid()); err != nil {
		return
	}
	if err = file.Truncate(int64(fileLen)); err != nil {
		return
	}
	err = file.Sync()
	return
}

// ReadPid reads process id from file and returns pid.
// If unable read from a file, returns error.
func (file *LockFile) ReadPid() (int, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, xerr.NewWM(err, "seek 0")
	}

	var pid int
	if _, err := fmt.Fscan(file, &pid); err != nil {
		return 0, xerr.NewWM(err, "scan pid")
	}

	return pid, nil
}

// Remove removes lock, closes and removes an open file.
func (file *LockFile) Remove() error {
	defer file.Close()

	if err := file.Unlock(); err != nil {
		return err
	}

	return os.Remove(file.Name())
}
