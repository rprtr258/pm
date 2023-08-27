package daemon

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-faster/tail"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
)

func Logs(ctx context.Context, follow bool) error {
	stat, errStat := os.Stat(_fileLog)
	if errStat != nil {
		return xerr.NewWM(errStat, "stat log file", xerr.Fields{"file": _fileLog})
	}

	const _defaultOffset = 10000

	t := tail.File(_fileLog, tail.Config{
		Location: &tail.Location{
			Offset: -fun.Min(stat.Size(), _defaultOffset),
			Whence: io.SeekEnd,
		},
		NotifyTimeout: 1 * time.Minute,
		Follow:        follow,
		BufferSize:    64 * 1024, //nolint:gomnd // 64 kb
		Logger:        nil,
		Tracker:       nil,
	})

	if err := t.Tail(ctx, func(ctx context.Context, l *tail.Line) error {
		fmt.Println(string(l.Data))
		return nil
	}); err != nil {
		return xerr.NewWM(err, "tail daemon logs")
	}

	return nil
}
