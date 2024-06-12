package logrotation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shoenig/test"
)

func TestNewFile(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)

	w := New(Config{
		Filename: fileLog(dir),
		clock:    useClock().Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)

	assertFileContent(t, fileLog(dir), b)
	assertFileCount(t, dir, 1)
}

func TestOpenExisting(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)

	filename := fileLog(dir)
	data := "foo!"
	test.NoError(t, os.WriteFile(filename, []byte(data), 0o644))
	assertFileContent(t, filename, data)

	w := New(Config{
		Filename: filename,
		clock:    useClock().Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)

	// make sure file got appended
	assertFileContent(t, filename, data+b)
	// make sure no other files were created
	assertFileCount(t, dir, 1)
}

func TestWriteTooLong(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)

	w := New(Config{
		Filename: fileLog(dir),
		MaxSize:  5,
		clock:    useClock().Now,
	})
	defer w.Close()

	b := []byte("booooooooooooooo!")
	n, err := w.Write(b)
	test.EqOp(t, 0, n)
	test.EqOp(t, fmt.Sprintf("write length %d exceeds maximum file size %d", len(b), w.maxSize), err.Error())

	_, err = os.Stat(fileLog(dir))
	test.ErrorIs(t, err, os.ErrNotExist, test.Sprint("File exists, but should not have been created"))
}

func TestMakeLogDir(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)
	filename := fileLog(dir)

	w := New(Config{
		Filename: filename,
		clock:    useClock().Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)

	assertFileContent(t, fileLog(dir), b)
	assertFileCount(t, dir, 1)
}

func TestDefaultFilename(t *testing.T) {
	t.Parallel()

	dir := os.TempDir()
	filename := filepath.Join(dir, filepath.Base(os.Args[0])+"-logrotation.log")
	defer os.Remove(filename)

	w := New(Config{
		clock: useClock().Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)

	assertFileContent(t, filename, b)
}

func TestAutoRotate(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)
	clock := useClock()
	filename := fileLog(dir)

	w := New(Config{
		Filename: filename,
		MaxSize:  10,
		clock:    clock.Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)

	assertFileContent(t, filename, b)
	assertFileCount(t, dir, 1)

	clock.advance()

	b2 := "foooooo!"
	assertWrite(t, w, b2)

	// old logfile should be moved aside and main logfile should have only last write in it
	assertFileContent(t, filename, b2)
	// backup file will use current fake time and have old contents
	assertFileContent(t, fileBackup(dir, clock.Now()), b)
	assertFileCount(t, dir, 2)
}

func TestFirstWriteRotate(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)
	clock := useClock()
	filename := fileLog(dir)

	w := New(Config{
		Filename: filename,
		MaxSize:  10,
		clock:    clock.Now,
	})
	defer w.Close()

	start := "boooooo!"
	test.NoError(t, os.WriteFile(filename, []byte(start), 0o600))

	clock.advance()

	// this would make us rotate
	b := "fooo!"
	assertWrite(t, w, b)

	assertFileContent(t, filename, b)
	assertFileContent(t, fileBackup(dir, clock.Now()), start)
	assertFileCount(t, dir, 2)
}

func TestMaxBackups(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)
	clock := useClock()
	filename := fileLog(dir)

	w := New(Config{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 1,
		clock:      clock.Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)

	assertFileContent(t, filename, b)
	assertFileCount(t, dir, 1)

	clock.advance()

	// this will put us over max
	b2 := "foooooo!"
	assertWrite(t, w, b2)

	// this will use new fake time
	secondFilename := fileBackup(dir, clock.Now())
	assertFileContent(t, secondFilename, b)
	// make sure old file still exists with same content
	assertFileContent(t, filename, b2)
	assertFileCount(t, dir, 2)

	clock.advance()

	// this will make us rotate again
	b3 := "baaaaaar!"
	assertWrite(t, w, b3)

	// this will use new fake time
	thirdFilename := fileBackup(dir, clock.Now())
	assertFileContent(t, thirdFilename, b2)
	assertFileContent(t, filename, b3)

	// we need to wait little bit since files get deleted on different goroutine
	<-time.After(time.Millisecond * 10)

	// should only have two files in dir still
	assertFileCount(t, dir, 2)
	// second file name should still exist
	assertFileContent(t, thirdFilename, b2)
	// should have deleted first backup
	test.FileNotExists(t, secondFilename)

	// now test that we don't delete directories or non-logfile files
	clock.advance()

	// create file that is close to but different from logfile name
	// It shouldn't get caught by our deletion filters.
	notlogfile := fileLog(dir) + ".foo"
	test.NoError(t, os.WriteFile(notlogfile, []byte("data"), 0o644))

	// Make directory that exactly matches our log file filters... it still
	// shouldn't get caught by deletion filter since it's directory.
	notlogfiledir := fileBackup(dir, clock.Now())
	test.NoError(t, os.Mkdir(notlogfiledir, 0o700))

	clock.advance()

	// this will use new fake time
	fourthFilename := fileBackup(dir, clock.Now())

	// Create log file that is/was being compressed - this should
	// not be counted since both compressed and uncompressed
	// log files still exist.
	compLogFile := fourthFilename + compressSuffix
	test.NoError(t, os.WriteFile(compLogFile, []byte("compress"), 0o644))

	// this will make us rotate again
	b4 := "baaaaaaz!"
	assertWrite(t, w, b4)

	assertFileContent(t, fourthFilename, b3)
	assertFileContent(t, compLogFile, "compress")

	// we need to wait little bit since files get deleted on different goroutine
	<-time.After(time.Millisecond * 10)

	// we should have four things in directory now - 2 log files, not log file, and directory
	assertFileCount(t, dir, 5)
	// third file name should still exist
	assertFileContent(t, filename, b4)
	assertFileContent(t, fourthFilename, b3)
	// should have deleted first filename
	test.FileNotExists(t, thirdFilename)
	// not-a-logfile should still exist
	test.FileExists(t, notlogfile)
	// directory
	test.DirExists(t, notlogfiledir)
}

func TestCleanupExistingBackups(t *testing.T) {
	t.Parallel()

	// test that if we start with more backup files than we're supposed to have
	// in total, that extra ones get cleaned up when we rotate.

	dir := useTempDir(t)
	clock := useClock()

	// make 3 backup files

	data := []byte("data")
	backup := fileBackup(dir, clock.Now())
	test.NoError(t, os.WriteFile(backup, data, 0o644))

	clock.advance()

	backup = fileBackup(dir, clock.Now())
	test.NoError(t, os.WriteFile(backup+compressSuffix, data, 0o644))

	clock.advance()

	backup = fileBackup(dir, clock.Now())
	test.NoError(t, os.WriteFile(backup, data, 0o644))

	// now create primary log file with some data
	filename := fileLog(dir)
	test.NoError(t, os.WriteFile(filename, data, 0o644))

	w := New(Config{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 1,
		clock:      clock.Now,
	})
	defer w.Close()

	clock.advance()

	assertWrite(t, w, "foooooo!")

	// we need to wait little bit since files get deleted on different goroutine
	<-time.After(time.Millisecond * 10)

	// now we should only have 2 files left - primary and one backup
	assertFileCount(t, dir, 2)
}

func TestMaxAge(t *testing.T) {
	t.Parallel()

	clock := useClock()
	dir := useTempDir(t)
	filename := fileLog(dir)

	w := New(Config{
		Filename: filename,
		MaxSize:  10,
		MaxAge:   time.Hour * 24,
		clock:    clock.Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)
	assertFileContent(t, filename, b)
	assertFileCount(t, dir, 1)

	// two days later
	clock.advance()

	b2 := "foooooo!"
	assertWrite(t, w, b2)
	assertFileContent(t, fileBackup(dir, clock.Now()), b)

	// we need to wait little bit since files get deleted on different goroutine
	<-time.After(10 * time.Millisecond)

	// We should still have 2 log files, since most recent backup was just
	// created.
	assertFileCount(t, dir, 2)
	assertFileContent(t, filename, b2)
	// we should have deleted old file due to being too old
	assertFileContent(t, fileBackup(dir, clock.Now()), b)

	// two days later
	clock.advance()

	b3 := "baaaaar!"
	assertWrite(t, w, b3)
	assertFileContent(t, fileBackup(dir, clock.Now()), b2)

	// we need to wait little bit since files get deleted on different goroutine
	<-time.After(10 * time.Millisecond)

	// We should have 2 log files - main log file, and most recent
	// backup. earlier backup is past cutoff and should be gone.
	assertFileCount(t, dir, 2)
	assertFileContent(t, filename, b3)
	// we should have deleted old file due to being too old
	assertFileContent(t, fileBackup(dir, clock.Now()), b2)
}

func TestOldLogFiles(t *testing.T) {
	t.Parallel()

	clock := useClock()
	dir := useTempDir(t)

	filename := fileLog(dir)
	data := []byte("data")
	test.NoError(t, os.WriteFile(filename, data, 0o7))

	// this gives us time with same precision as time we get from timestamp in name
	t1, err := time.Parse(backupTimeFormat, clock.Now().UTC().Format(backupTimeFormat))
	test.NoError(t, err)

	backup := fileBackup(dir, clock.Now())
	test.NoError(t, os.WriteFile(backup, data, 0o7))

	clock.advance()

	t2, err := time.Parse(backupTimeFormat, clock.Now().UTC().Format(backupTimeFormat))
	test.NoError(t, err)

	backup2 := fileBackup(dir, clock.Now())
	test.NoError(t, os.WriteFile(backup2, data, 0o7))

	w := New(Config{
		Filename: filename,
		clock:    clock.Now,
	})
	files, err := w.oldLogFiles()
	test.NoError(t, err)
	test.EqOp(t, 2, len(files))

	// should be sorted by newest file first, which would be t2
	test.EqOp(t, t2, files[0].timestamp)
	test.EqOp(t, t1, files[1].timestamp)
}

func TestTimeFromName(t *testing.T) {
	t.Parallel()

	prefix, ext := prefixAndExt("/var/log/myfoo/foo.log")

	// happy cases
	for filename, want := range map[string]time.Time{
		"foo-2014-05-04T14-44-33.555.log": time.Date(2014, 5, 4, 14, 44, 33, 555000000, time.UTC),
	} {
		got, err := timeFromName(filename, prefix, ext)
		test.NoError(t, err)
		test.EqOp(t, want, got)
	}

	// fail cases
	for _, filename := range []string{
		"foo-2014-05-04T14-44-33.555", // no extension
		"2014-05-04T14-44-33.555.log", // no prefix
		"foo.log",                     // no timestamp
	} {
		_, err := timeFromName(filename, prefix, ext)
		test.Error(t, err)
	}
}

func TestLocalTime(t *testing.T) {
	t.Parallel()

	clock := useClock()
	dir := useTempDir(t)
	filename := fileLog(dir)

	w := New(Config{
		Filename:  filename,
		MaxSize:   10,
		LocalTime: true,
		clock:     clock.Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)

	b2 := "fooooooo!"
	assertWrite(t, w, b2)

	assertFileContent(t, fileBackupLocal(dir, clock.Now()), b)
	assertFileContent(t, filename, b2)
}

func TestRotate(t *testing.T) {
	t.Parallel()

	dir := useTempDir(t)
	clock := useClock()
	filename := fileLog(dir)

	w := New(Config{
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
		clock:      clock.Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)

	assertFileContent(t, filename, b)
	assertFileCount(t, dir, 1)

	clock.advance()

	test.NoError(t, w.Rotate())
	// we need to wait little bit since files get deleted on different goroutine
	<-time.After(10 * time.Millisecond)

	filename2 := fileBackup(dir, clock.Now())
	assertFileContent(t, filename2, b)
	assertFileContent(t, filename, "")
	assertFileCount(t, dir, 2)

	clock.advance()

	test.NoError(t, w.Rotate())
	// we need to wait little bit since files get deleted on different goroutine
	<-time.After(10 * time.Millisecond)

	filename3 := fileBackup(dir, clock.Now())
	assertFileContent(t, filename3, "")
	assertFileContent(t, filename, "")
	assertFileCount(t, dir, 2)

	b2 := "foooooo!"
	assertWrite(t, w, b2)

	// this will use new fake time
	assertFileContent(t, filename, b2)
}

func TestCompressOnRotate(t *testing.T) {
	t.Parallel()

	clock := useClock()
	dir := useTempDir(t)
	filename := fileLog(dir)

	w := New(Config{
		Compress: true,
		Filename: filename,
		MaxSize:  10,
		clock:    clock.Now,
	})
	defer w.Close()

	b := "boo!"
	assertWrite(t, w, b)
	assertFileContent(t, filename, b)
	assertFileCount(t, dir, 1)

	clock.advance()

	test.NoError(t, w.Rotate())

	// old logfile should be moved aside and main logfile should have nothing in it
	assertFileContent(t, filename, "")

	// we need to wait little bit since files get compressed on different goroutine
	<-time.After(300 * time.Millisecond)

	// compressed version of log file should now exist and original should have been removed
	assertFileContent(t, fileBackup(dir, clock.Now())+compressSuffix, useGzip(t, b))
	test.FileNotExists(t, fileBackup(dir, clock.Now()))

	assertFileCount(t, dir, 2)
}

func TestCompressOnResume(t *testing.T) {
	t.Parallel()

	clock := useClock()
	dir := useTempDir(t)
	filename := fileLog(dir)

	w := New(Config{
		Compress: true,
		Filename: filename,
		MaxSize:  10,
		clock:    clock.Now,
	})
	defer w.Close()

	// create backup file and empty "compressed" file
	filename2 := fileBackup(dir, clock.Now())
	b := "foo!"
	test.NoError(t, os.WriteFile(filename2, []byte(b), 0o644))
	test.NoError(t, os.WriteFile(filename2+compressSuffix, []byte{}, 0o644))

	clock.advance()

	b2 := "boo!"
	assertWrite(t, w, b2)
	assertFileContent(t, filename, b2)

	// we need to wait little bit since files get compressed on different goroutine
	<-time.After(300 * time.Millisecond)

	// write should have started compression - compressed version of
	// log file should now exist and original should have been removed.
	assertFileContent(t, filename2+compressSuffix, useGzip(t, b))
	test.FileNotExists(t, filename2)

	assertFileCount(t, dir, 2)
}
