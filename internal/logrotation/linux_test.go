package logrotation

import (
	"os"
	"testing"
	"time"

	"github.com/shoenig/test"
)

func TestMaintainMode(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)
	clock := useClock()
	filename := fileLog(dir)

	mode := os.FileMode(0600)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, mode)
	test.NoError(t, err)
	f.Close()

	w := New(Config{
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100,
		clock:      clock.Now,
	})
	defer w.Close()

	assertWrite(t, w, "boo!")

	clock.advance()

	test.NoError(t, w.Rotate())

	info, err := os.Stat(filename)
	test.NoError(t, err)
	test.EqOp(t, mode, info.Mode())

	filename2 := fileBackup(dir, clock.Now())

	info2, err := os.Stat(filename2)
	test.NoError(t, err)
	test.EqOp(t, mode, info2.Mode())
}

func TestMaintainOwner(t *testing.T) {
	t.Parallel()

	fakeFS := useFs()
	dir := useTempDir(t)
	clock := useClock()
	filename := fileLog(dir)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	test.NoError(t, err)
	f.Close()

	w := New(Config{
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
		fs:         fakeFS,
		clock:      clock.Now,
	})
	defer w.Close()

	assertWrite(t, w, "boo!")

	clock.advance()

	test.NoError(t, w.Rotate())

	test.EqOp(t, 555, fakeFS.files[filename].uid)
	test.EqOp(t, 666, fakeFS.files[filename].gid)
}

func TestCompressMaintainMode(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)
	clock := useClock()
	filename := fileLog(dir)

	mode := os.FileMode(0600)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, mode)
	test.NoError(t, err)
	f.Close()

	w := New(Config{
		Compress:   true,
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
		clock:      clock.Now,
	})
	defer w.Close()

	assertWrite(t, w, "boo!")

	clock.advance()

	test.NoError(t, w.Rotate())
	// we need to wait little bit since files get compressed in different goroutine
	<-time.After(10 * time.Millisecond)

	info, err := os.Stat(filename)
	test.NoError(t, err)
	test.EqOp(t, mode, info.Mode())

	// compressed version of log file should now exist with correct mode
	filename2 := fileBackup(dir, clock.Now())

	info2, err := os.Stat(filename2 + compressSuffix)
	test.NoError(t, err)
	test.EqOp(t, mode, info2.Mode())
}

func TestCompressMaintainOwner(t *testing.T) {
	t.Parallel()

	fakeFS := useFs()
	clock := useClock()
	dir := useTempDir(t)
	filename := fileLog(dir)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	test.NoError(t, err)
	f.Close()

	w := New(Config{
		Compress:   true,
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
		fs:         fakeFS,
		clock:      clock.Now,
	})
	defer w.Close()

	assertWrite(t, w, "boo!")

	clock.advance()

	test.NoError(t, w.Rotate())
	// we need to wait little bit since files get compressed on different goroutine
	<-time.After(10 * time.Millisecond)

	// compressed version of log file should now exist with correct owner
	filename2 := fileBackup(dir, clock.Now())
	test.EqOp(t, 555, fakeFS.files[filename2+compressSuffix].uid)
	test.EqOp(t, 666, fakeFS.files[filename2+compressSuffix].gid)
}
