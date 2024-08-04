package logrotation

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/shoenig/test"
	"github.com/spf13/afero"
)

type fakeFile struct {
	uid, gid int
}

type fakeFS struct {
	afero.Fs
	files map[string]fakeFile
	mu    sync.Mutex
}

func useFs() *fakeFS {
	return &fakeFS{
		files: make(map[string]fakeFile),

		// not set
		mu: sync.Mutex{},
		Fs: nil,
	}
}

func (fs *fakeFS) Open(name string) (afero.File, error) {
	return afero.NewOsFs().Open(name)
}

func (fs *fakeFS) Chown(name string, uid, gid int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.files[name] = fakeFile{uid: uid, gid: gid}
	return nil
}

func (fs *fakeFS) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}

	stat, _ := info.Sys().(*syscall.Stat_t)
	stat.Uid = 555
	stat.Gid = 666
	return info, nil
}

type fakeClock struct {
	now time.Time
}

func useClock() *fakeClock          { return &fakeClock{time.Now()} }
func (c *fakeClock) Now() time.Time { return c.now }

// advance sets fake "current time" to two days later
func (c *fakeClock) advance() { c.now = c.now.Add(time.Hour * 24 * 2) }

func assertWrite(tb testing.TB, l *Writer, s string) {
	tb.Helper()

	n, err := l.Write([]byte(s))
	test.NoError(tb, err)
	test.EqOp(tb, len(s), n)
}

// useTempDir creates file with semi-unique name in OS temp directory.
// It should be based on name of test, to keep parallel tests from
// colliding, and must be cleaned up after test is finished.
func useTempDir(tb testing.TB) string {
	tb.Helper()

	now := time.Now().Format(backupTimeFormat)
	dir := filepath.Join(os.TempDir(), tb.Name()+now)
	test.Nil(tb, os.Mkdir(dir, 0o700))
	tb.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func useGzip(tb testing.TB, s string) string {
	tb.Helper()

	var bc bytes.Buffer
	gz := gzip.NewWriter(&bc)
	_, err := gz.Write([]byte(s))
	test.NoError(tb, err)
	test.NoError(tb, gz.Close())
	return bc.String()
}

// assertFileContent checks that given file exists and has correct content.
func assertFileContent(tb testing.TB, path, content string) {
	tb.Helper()

	b, err := os.ReadFile(path)
	test.NoError(tb, err)
	test.Eq(tb, content, string(b))
}

// assertFileCount checks that number of files in directory is exp
func assertFileCount(tb testing.TB, dir string, exp int) {
	tb.Helper()

	files, err := os.ReadDir(dir)
	test.NoError(tb, err)
	// make sure no other files were created
	test.EqOp(tb, exp, len(files))
}

// fileLog returns log file name in given directory for current fake time
func fileLog(dir string) string {
	return filepath.Join(dir, "foobar.log")
}

func fileBackup(dir string, now time.Time) string {
	return filepath.Join(dir, "foobar-"+now.UTC().Format(backupTimeFormat)+".log")
}

func fileBackupLocal(dir string, now time.Time) string {
	return filepath.Join(dir, "foobar-"+now.Format(backupTimeFormat)+".log")
}
