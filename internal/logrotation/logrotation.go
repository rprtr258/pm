// Package assumes that only one process is writing to output files.
// Using same configuration from multiple processes on same
// machine will result in improper behavior.
package logrotation

import (
	"cmp"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

const (
	backupTimeFormat = "2006-01-02T15-04-05.000"
	compressSuffix   = ".gz"
)

// ensure we always implement io.WriteCloser
var _ io.WriteCloser = (*Writer)(nil)

// Writer is an io.WriteCloser that writes to specified filename.
//
// Writer opens or creates logfile on first Write. If file exists and
// is less than MaxSize megabytes, it will open and append to that file.
// If file exists and its size is >= MaxSize megabytes, file is renamed
// by putting current time in a timestamp in name immediately before the
// file's extension (or end of filename if there's no extension). A new
// log file is then created using original filename.
//
// Whenever a write would cause current log file exceed MaxSize megabytes,
// current file is closed, renamed, and a new log file created with the
// original name. Thus, filename you give Writer is always "current" log
// file.
//
// Backups use log file name given to Writer, in form
// `name-timestamp.ext` where name is filename without extension,
// timestamp is time at which log was rotated formatted with the
// time.Time format of `2006-01-02T15-04-05.000` and extension is the
// original extension. For example, if your Writer.Filename is
// `/var/log/foo/server.log`, a backup created at 6:30pm on Nov 11 2016 would
// use filename `/var/log/foo/server-2016-11-04T18-30-00.000.log`
//
// # Cleaning Up Old Log Files
//
// Whenever a new logfile gets created, old log files may be deleted. most
// recent files according to encoded timestamp will be retained, up to a
// number equal to MaxBackups (or all of them if MaxBackups is 0). Any files
// with an encoded timestamp older than MaxAge days are deleted, regardless of
// MaxBackups. Note that time encoded in timestamp is rotation
// time, which may differ from last time that file was written to.
//
// If MaxBackups and MaxAge are both 0, no old log files will be deleted.
type Writer struct {
	// filename is file to write logs to. Backup log files will be retained
	// in same directory.
	// It uses <processname>-logrotation.log in os.TempDir() if empty.
	filename string

	// maxSize is maximum size in bytes of log file before it gets rotated.
	// It defaults to 100 megabytes.
	maxSize int64

	// maxAge is maximum duration to retain old log files based on the
	// timestamp encoded in their filename. Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc.
	// default is not to remove old log files based on age.
	maxAge time.Duration

	// maxBackups is maximum number of old log files to retain.
	// Default is to retain all old log files (though MaxAge may still cause them to get deleted).
	maxBackups int

	// isLocalTime determines if time used for formatting timestamps in
	// backup files is computer's local time.
	// Default is to use UTC time.
	isLocalTime bool

	// compress determines if rotated log files should be compressed using gzip.
	// Default is not to perform compression.
	compress bool

	size int64
	file *os.File
	mu   sync.Mutex

	millCh    chan bool
	startMill sync.Once

	fs    afero.Fs
	clock clock
}

type Config struct {
	// Filename is file to write logs to. Backup log files will be retained
	// in same directory.
	// It uses <processname>-logrotation.log in os.TempDir() if empty.
	Filename string

	// MaxSize is maximum size in bytes of log file before it gets rotated.
	// It defaults to 100 megabytes.
	MaxSize int64

	// MaxAge is maximum duration to retain old log files based on the
	// timestamp encoded in their filename. Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc.
	// default is not to remove old log files based on age.
	MaxAge time.Duration

	// MaxBackups is maximum number of old log files to retain.
	// Default is to retain all old log files (though MaxAge may still cause them to get deleted).
	MaxBackups int

	// LocalTime determines if time used for formatting timestamps in
	// backup files is computer's local time.
	// Default is to use UTC time.
	LocalTime bool

	// Compress determines if rotated log files should be compressed using gzip.
	// Default is not to perform compression.
	Compress bool

	fs    afero.Fs
	clock clock
}

func New(cfg Config) *Writer {
	clock := time.Now
	if cfg.clock != nil {
		clock = cfg.clock
	}

	fs := afero.NewOsFs()
	if cfg.fs != nil {
		fs = cfg.fs
	}

	return &Writer{
		filename: cmp.Or(
			cfg.Filename,
			filepath.Join(os.TempDir(), filepath.Base(os.Args[0])+"-logrotation.log"),
		),
		maxSize: cmp.Or(
			cfg.MaxSize,
			int64(100*1024*1024), // 100 MiB
		),
		maxAge:      cfg.MaxAge,
		maxBackups:  cfg.MaxBackups,
		isLocalTime: cfg.LocalTime,
		compress:    cfg.Compress,
		fs:          fs,
		clock:       clock,

		// not set
		size:      0,
		file:      nil,
		mu:        sync.Mutex{},
		millCh:    nil,
		startMill: sync.Once{},
	}
}

// Write implements io.Writer. If a write would cause log file to be larger
// than MaxSize, file is closed, renamed to include a timestamp of the
// current time, and a new log file is created using original log file name.
// If length of write is greater than MaxSize, an error is returned.
func (l *Writer) Write(b []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	writeLen := int64(len(b))
	if writeLen > l.maxSize {
		return 0, fmt.Errorf("write length %d exceeds maximum file size %d", writeLen, l.maxSize)
	}

	if l.file == nil {
		if err := l.openExistingOrNew(len(b)); err != nil {
			return 0, err
		}
	}

	if l.size+writeLen > l.maxSize {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := l.file.Write(b)
	l.size += int64(n)

	return n, err
}

// Close implements io.Closer, and closes current logfile.
func (l *Writer) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.close()
}

// close closes file if it is open.
func (l *Writer) close() error {
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

// Rotate causes Logger to close existing log file and immediately create a new one.
// This is a helper function for applications that want to initiate
// rotations outside of normal rotation rules, such as in response to SIGHUP.
// After rotating, this initiates compression and removal of old log files according to configuration.
func (l *Writer) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.rotate()
}

// rotate closes current file, moves it aside with a timestamp in name,
// (if it exists), opens a new file with original filename, and then runs
// post-rotation processing and removal.
func (l *Writer) rotate() error {
	if err := l.close(); err != nil {
		return err
	}

	if err := l.openNew(); err != nil {
		return err
	}

	l.mill()
	return nil
}

func chown(fs afero.Fs, name string, info os.FileInfo) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	f.Close()

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("get file owner and group")
	}

	return fs.Chown(name, int(stat.Uid), int(stat.Gid))
}

// backupName creates a new filename from given name, inserting a timestamp
// between filename and extension
func backupName(now time.Time, name string) string {
	dir := filepath.Dir(name)
	filename := filepath.Base(name)
	ext := filepath.Ext(filename)
	prefix := filename[:len(filename)-len(ext)]

	timestamp := now.Format(backupTimeFormat)
	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", prefix, timestamp, ext))
}

// openNew opens new log file for writing, moving any old log file out of way.
// This methods assumes file has already been closed.
func (l *Writer) openNew() error {
	if err := os.MkdirAll(l.dir(), 0o755); err != nil {
		return fmt.Errorf("can't make directories for new logfile: %w", err)
	}

	name := l.filename
	mode := os.FileMode(0o600)
	if stat, err := l.fs.Stat(name); err == nil {
		// copy mode off old logfile
		mode = stat.Mode()

		now := l.clock()
		if !l.isLocalTime {
			now = now.UTC()
		}

		// move existing file
		newname := backupName(now, name)
		if err := os.Rename(name, newname); err != nil {
			return fmt.Errorf("can't rename log file: %w", err)
		}

		if err := chown(l.fs, name, stat); err != nil {
			return err
		}
	}

	// we use truncate here because this should only get called when we've moved
	// file ourselves. if someone else creates file in meantime,
	// just wipe out contents.
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %w", err)
	}

	l.file = f
	l.size = 0
	return nil
}

// openExistingOrNew opens logfile if it exists and if current write would not put it over MaxSize.
// If there is no such file or write would put it over MaxSize, a new file is created.
func (l *Writer) openExistingOrNew(writeLen int) error {
	l.mill()

	filename := l.filename
	info, err := l.fs.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return l.openNew()
		}
		return fmt.Errorf("error getting log file info: %w", err)
	}

	if info.Size()+int64(writeLen) >= l.maxSize {
		return l.rotate()
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		// if we fail to open old log file for some reason, just ignore it and open new log file
		return l.openNew()
	}

	l.file = file
	l.size = info.Size()
	return nil
}

// logInfo is convenience struct to return filename and its embedded timestamp
type logInfo struct {
	timestamp time.Time
	os.FileInfo
}

// millRunOnce performs compression and removal of stale log files.
// Log files are compressed if enabled via configuration and old log
// files are removed, keeping at most l.MaxBackups files, as long as
// none of them are older than MaxAge.
func (l *Writer) millRunOnce() error {
	if l.maxBackups == 0 && l.maxAge == 0 && !l.compress {
		return nil
	}

	files, err := l.oldLogFiles()
	if err != nil {
		return err
	}

	var remove []logInfo
	if l.maxBackups > 0 && l.maxBackups < len(files) {
		preserved := map[string]struct{}{}
		var remaining []logInfo
		for _, f := range files {
			// only count uncompressed log file or the compressed log file, not both
			fn := strings.TrimSuffix(f.Name(), compressSuffix)
			preserved[fn] = struct{}{}

			if len(preserved) > l.maxBackups {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}
	if l.maxAge > 0 {
		cutoff := l.clock().Add(-l.maxAge)

		var remaining []logInfo
		for _, f := range files {
			if f.timestamp.Before(cutoff) {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}

	var compress []logInfo
	if l.compress {
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), compressSuffix) {
				compress = append(compress, f)
			}
		}
	}

	for _, f := range remove {
		errRemove := os.Remove(filepath.Join(l.dir(), f.Name()))
		if err == nil && errRemove != nil {
			err = errRemove
		}
	}
	for _, f := range compress {
		fn := filepath.Join(l.dir(), f.Name())
		errCompress := compressLogFile(l.fs, fn, fn+compressSuffix)
		if err == nil && errCompress != nil {
			err = errCompress
		}
	}

	return err
}

// mill performs post-rotation compression and removal of stale log files,
// starting mill goroutine if necessary.
func (l *Writer) mill() {
	l.startMill.Do(func() {
		l.millCh = make(chan bool, 1)
		// manage post-rotation compression and removal of old log files
		go func() {
			for range l.millCh {
				// what am I going to do, log this?
				_ = l.millRunOnce()
			}
		}()
	})

	select {
	case l.millCh <- true:
	default:
	}
}

func readDir(dirname string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}

	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// oldLogFiles returns list of backup log files stored in same
// directory as current log file, sorted by ModTime
func (l *Writer) oldLogFiles() ([]logInfo, error) {
	files, err := readDir(l.dir())
	if err != nil {
		return nil, fmt.Errorf("can't read log file directory: %w", err)
	}

	prefix, ext := prefixAndExt(l.filename)

	logFiles := []logInfo{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if t, err := timeFromName(f.Name(), prefix, ext); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
			continue
		}
		if t, err := timeFromName(f.Name(), prefix, ext+compressSuffix); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
			continue
		}
		// error parsing means that suffix at end was not generated
		// by package, and therefore it's not a backup file.
	}
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].timestamp.After(logFiles[j].timestamp)
	})
	return logFiles, nil
}

// timeFromName extracts formatted time from filename by stripping off filename's prefix and extension.
// This prevents someone's filename from confusing time.parse.
func timeFromName(filename, prefix, ext string) (time.Time, error) {
	if !strings.HasPrefix(filename, prefix) {
		return time.Time{}, errors.New("mismatched prefix")
	}
	if !strings.HasSuffix(filename, ext) {
		return time.Time{}, errors.New("mismatched extension")
	}
	ts := filename[len(prefix) : len(filename)-len(ext)]
	return time.Parse(backupTimeFormat, ts)
}

// dir returns directory for current filename
func (l *Writer) dir() string {
	return filepath.Dir(l.filename)
}

// prefixAndExt returns filename part and extension part from Logger's filename
func prefixAndExt(filename string) (prefix, ext string) {
	base := filepath.Base(filename)
	ext = filepath.Ext(base)
	prefix = base[:len(base)-len(ext)] + "-"
	return prefix, ext
}

// compressLogFile compresses given log file, removing uncompressed log file if successful
func compressLogFile(fs afero.Fs, src, dst string) (err error) {
	f, err := fs.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	stat, err := fs.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	if errChown := chown(fs, dst, stat); errChown != nil {
		return fmt.Errorf("failed to chown compressed log file: %w", errChown)
	}

	// If this file already exists, we presume it was created by
	// a previous attempt to compress log file.
	gzf, errOpen := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, stat.Mode())
	if errOpen != nil {
		return fmt.Errorf("failed to open compressed log file: %w", errOpen)
	}
	defer gzf.Close()

	gz := gzip.NewWriter(gzf)
	defer gz.Close()

	defer func() {
		if err != nil {
			os.Remove(dst)
			err = fmt.Errorf("failed to compress log file: %w", err)
		}
	}()

	if _, err := io.Copy(gz, f); err != nil {
		return err
	}

	if err := os.Remove(src); err != nil {
		return err
	}

	return nil
}
