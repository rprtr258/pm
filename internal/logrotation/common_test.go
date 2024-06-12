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
	return &fakeFS{files: make(map[string]fakeFile)}
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
	stat := info.Sys().(*syscall.Stat_t)
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

func assertWrite(t testing.TB, l *Writer, s string) {
	t.Helper()

	n, err := l.Write([]byte(s))
	test.NoError(t, err)
	test.EqOp(t, len(s), n)
}

// useTempDir creates file with semi-unique name in OS temp directory.
// It should be based on name of test, to keep parallel tests from
// colliding, and must be cleaned up after test is finished.
func useTempDir(t testing.TB) string {
	t.Helper()

	now := time.Now().Format(backupTimeFormat)
	dir := filepath.Join(os.TempDir(), t.Name()+now)
	test.Nil(t, os.Mkdir(dir, 0o700))
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func useGzip(t testing.TB, s string) string {
	t.Helper()

	var bc bytes.Buffer
	gz := gzip.NewWriter(&bc)
	_, err := gz.Write([]byte(s))
	test.NoError(t, err)
	test.NoError(t, gz.Close())
	return bc.String()
}

// assertFileContent checks that given file exists and has correct content.
func assertFileContent(t testing.TB, path string, content string) {
	t.Helper()

	b, err := os.ReadFile(path)
	test.NoError(t, err)
	test.Eq(t, content, string(b))
}

// assertFileCount checks that number of files in directory is exp
func assertFileCount(t testing.TB, dir string, exp int) {
	t.Helper()

	files, err := os.ReadDir(dir)
	test.NoError(t, err)
	// make sure no other files were created
	test.EqOp(t, exp, len(files))
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
