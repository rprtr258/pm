package daemon

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	filename                = os.TempDir() + "/test.lock"
	fileperm    os.FileMode = 0o644
	invalidname             = "/x/y/unknown"
)

func TestCreatePidFile(t *testing.T) {
	_, err := CreatePidFile(invalidname, fileperm)
	assert.Error(t, err, "CreatePidFile(): Error was not detected on invalid name")

	lock, err := CreatePidFile(filename, fileperm)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, lock.Remove())
	}()

	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	assert.Equal(t, string(data), fmt.Sprint(os.Getpid()))

	file, err := os.OpenFile(filename, os.O_RDONLY, fileperm)
	assert.NoError(t, err)

	err = NewLockFile(file).WritePid()
	assert.Error(t, err, "WritePid(): Error was not detected on invalid permissions")
}

func TestNewLockFile(test *testing.T) {
	lock := NewLockFile(os.NewFile(1001, ""))
	err := lock.Remove()
	if err == nil {
		test.Fatal("Remove(): Error was not detected on invalid fd")
	}
	err = lock.WritePid()
	if err == nil {
		test.Fatal("WritePid(): Error was not detected on invalid fd")
	}
}

func TestReadPid(t *testing.T) {
	lock, err := CreatePidFile(filename, fileperm)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, lock.Remove())
	}()

	pid, err := lock.ReadPid()
	assert.NoError(t, err, "ReadPid(): Unable read pid from file")

	assert.Equal(t, pid, os.Getpid(), "Pid not equal real pid")
}

func TestLockFileLock(t *testing.T) {
	lock1, err := OpenLockFile(filename, fileperm)
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, lock1.Lock())
	defer func() {
		assert.NoError(t, lock1.Remove())
	}()

	lock2, err := OpenLockFile(filename, fileperm)
	assert.NoError(t, err)

	if runtime.GOOS == "solaris" {
		// Solaris does not see a double lock attempt by the same process as failure.
		assert.NoError(t, lock2.Lock(), "To lock file more than once must be unavailable")
	} else {
		assert.ErrorContains(t, lock2.Lock(), "would block", "To lock file more than once must be unavailable")
	}
}
