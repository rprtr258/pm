package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nxadm/tail"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

type fileSize int64

const (
	_byte     fileSize = 1
	_kibibyte fileSize = 1024 * _byte

	_defaultLogsOffset = 1000 * _byte
)

type ProcLine struct {
	Line string
	Type core.LogType
	Err  error
}

func streamFile(
	ctx context.Context,
	logLinesCh chan ProcLine,
	procID core.PMID,
	logFile string,
	logLineType core.LogType,
	wg *sync.WaitGroup,
) error {
	stat, errStat := os.Stat(logFile)
	if errStat != nil {
		return errors.Wrapf(errStat, "stat log file %s", logFile)
	}

	tailer, err := tail.TailFile(logFile, tail.Config{
		Follow:        true,
		CompleteLines: true,
		ReOpen:        true,
		Location: &tail.SeekInfo{
			Whence: io.SeekEnd,
			Offset: -fun.Min(stat.Size(), int64(_defaultLogsOffset)),
		},
		Logger:      tail.DiscardingLogger,
		MustExist:   true,
		Poll:        false,
		Pipe:        false,
		MaxLineSize: 0,
		RateLimiter: nil,
	})
	if err != nil {
		return errors.Wrapf(err, "tail log, id=%s, file=%s", procID, logFile)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case line := <-tailer.Lines:
				logLinesCh <- ProcLine{
					Line: line.Text,
					Type: logLineType,
					Err:  line.Err,
				}
			}
		}
	}()

	return nil
}

func streamProcLogs(ctx context.Context, proc core.ProcStat) <-chan ProcLine {
	logLinesCh := make(chan ProcLine)
	go func() {
		var wg sync.WaitGroup
		if err := streamFile(
			ctx,
			logLinesCh,
			proc.ID,
			proc.StdoutFile,
			core.LogTypeStdout,
			&wg,
		); err != nil {
			log.Error().
				// Uint64("procID", proc.ID).
				Str("file", proc.StdoutFile).
				Err(err).
				Msg("failed to stream stdout log file")
		}
		if err := streamFile(
			ctx,
			logLinesCh,
			proc.ID,
			proc.StderrFile,
			core.LogTypeStderr,
			&wg,
		); err != nil {
			log.Error().
				// Uint64("procID", proc.ID).
				Str("file", proc.StderrFile).
				Err(err).
				Msg("failed to stream stderr log file")
		}
		wg.Wait()
		close(logLinesCh)
	}()
	return logLinesCh
}

// implLogs - watch for processes logs
func implLogs(ctx context.Context, proc core.ProcStat) <-chan core.LogLine {
	ctx, cancel := context.WithCancel(ctx)
	if proc.Status != core.StatusRunning {
		ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	}

	logsCh := streamProcLogs(ctx, proc)

	res := make(chan core.LogLine)
	go func() {
		defer close(res)
		defer cancel()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, ok := linuxprocess.StatPMID(linuxprocess.List(), proc.ID, app.EnvPMID); !ok {
					return
				}
			case line, ok := <-logsCh:
				if !ok {
					return
				}

				select {
				case <-ctx.Done():
					return
				case res <- core.LogLine{
					ProcID:   proc.ID,
					ProcName: proc.Name,
					Line:     line.Line,
					Type:     line.Type,
					Err:      line.Err,
				}:
				}
			}
		}
	}()
	return res
}

func getProcs(
	db db.Handle,
	rest, ids, names, tags []string,
	config *string,
) ([]core.ProcStat, error) {
	filterFunc := core.FilterFunc(
		core.WithGeneric(rest...),
		core.WithIDs(ids...),
		core.WithNames(names...),
		core.WithTags(tags...),
		core.WithAllIfNoFilters,
	)

	if config == nil {
		return listProcs(db).
			Filter(func(ps core.ProcStat) bool { return filterFunc(ps.Proc) }).
			ToSlice(), nil
	}

	configs, errLoadConfigs := core.LoadConfigs(*config)
	if errLoadConfigs != nil {
		return nil, errors.Wrapf(errLoadConfigs, "load configs: %v", *config)
	}

	procNames := fun.FilterMap[string](func(cfg core.RunConfig) (string, bool) {
		return cfg.Name.Unpack()
	}, configs...)

	return listProcs(db).
		Filter(func(proc core.ProcStat) bool { return fun.Contains(proc.Name, procNames...) }).
		Filter(func(ps core.ProcStat) bool { return filterFunc(ps.Proc) }).
		ToSlice(), nil
}

var (
	_barStdout = scuf.String("|", scuf.FgGreen)
	_barStderr = scuf.String("|", scuf.FgRed)
)

// TODO: cleanup logs files which are not bound to any existing process (in any status)
var _cmdLogs = func() *cobra.Command {
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "logs [name|tag|id]...",
		Short:             "watch for processes logs",
		GroupID:           "inspection",
		ValidArgsFunction: completeArgGenericSelector,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			appp, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrapf(errNewApp, "new app")
			}

			procs, err := getProcs(appp.DB, args, ids, names, tags, config)
			if err != nil {
				return errors.Wrapf(err, "get proc ids")
			}
			if len(procs) == 0 {
				fmt.Println("nothing to watch")
				return nil
			}

			var wg sync.WaitGroup
			mergedLogsCh := make(chan core.LogLine)
			for _, proc := range procs {
				logsCh := implLogs(ctx, proc)

				wg.Add(1)
				ch := logsCh
				go func() {
					defer wg.Done()
					for {
						select {
						case <-ctx.Done():
							return
						case v, ok := <-ch:
							if !ok {
								return
							}

							select {
							case <-ctx.Done():
								return
							case mergedLogsCh <- v:
							}
						}
					}
				}()
			}
			go func() {
				wg.Wait()
				close(mergedLogsCh)
			}()

			pad := 0
			for {
				select {
				case <-ctx.Done():
					return nil
				case line, ok := <-mergedLogsCh:
					if !ok {
						return nil
					}

					if line.Err != nil {
						line.Line = line.Err.Error()
					}

					lineColor := fun.Switch(line.Type, scuf.FgRed).
						Case(scuf.FgHiWhite, core.LogTypeStdout).
						Case(scuf.FgHiBlack, core.LogTypeStderr).
						End()

					barColor := fun.IF(line.Type == core.LogTypeStdout, _barStdout, _barStderr)

					pad = max(pad, len(line.ProcName))
					// {proc} {pad}{sep} {line}
					fmt.Println(
						scuf.String(line.ProcName, colorByID(line.ProcID)),
						strings.Repeat(" ", pad-len(line.ProcName)+1)+barColor,
						scuf.String(line.Line, lineColor),
					)
				}
			}
		},
	}
	//   .option('--json', 'json log output')
	//   .option('--format', 'formated log output')
	//   .option('--raw', 'raw output')
	//   .option('--err', 'only shows error output')
	//   .option('--out', 'only shows standard output')
	//   .option('--lines <n>', 'output the last N lines, instead of the last 15 by default')
	//   .option('--timestamp [format]', 'add timestamps (default format YYYY-MM-DD-HH:mm:ss)')
	//   .option('--highlight', 'enable highlighting')
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()

var colors = [...]scuf.Modifier{
	scuf.FgHiRed,
	scuf.FgHiGreen,
	scuf.FgHiYellow,
	scuf.FgHiBlue,
	scuf.FgHiMagenta,
	scuf.FgHiCyan,
	scuf.FgHiWhite,
}

func colorByID(id core.PMID) scuf.Modifier {
	x := 0
	for i := 0; i < len(id); i++ {
		x += int(id[i])
	}
	return colors[x%len(colors)]
}
