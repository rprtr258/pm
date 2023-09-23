package daemon

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

// TODO: SIGCHLD part not working, fix
func StartDeathCollector(ctx context.Context, ebus *eventbus.EventBus, db db.Handle) {
	c := make(chan os.Signal, 10) // arbitrary buffer size
	signal.Notify(c, syscall.SIGCHLD)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("context canceled, stopping...")
			return
		case <-ticker.C:
			for procID, proc := range db.GetProcs(core.WithAllIfNoFilters) {
				if proc.Status.Status != core.StatusRunning {
					continue
				}

				switch _, errStat := linuxprocess.ReadProcessStat(proc.Status.Pid); errStat {
				case nil:
					// process stat file exists hence process is still running
					continue
				case linuxprocess.ErrStatFileNotFound:
					log.Info().
						Int("pid", proc.Status.Pid).
						Msg("process seems to be stopped, updating status...")

					ebus.Publish(ctx, eventbus.NewPublishProcStopped(procID, -1, eventbus.EmitReasonDied))
				default:
					log.Warn().
						Err(errStat).
						Int("pid", proc.Status.Pid).
						Msg("read proc stat")
				}
			}
		case <-c:
			// wait for any of childs' death
			for {
				var status syscall.WaitStatus
				pid, errWait := syscall.Wait4(-1, &status, 0, nil)
				if pid < 0 {
					break
				}
				if errWait != nil {
					log.Error().Err(errWait).Msg("Wait4 failed")
					continue
				}

				log.Info().Int("pid", pid).Msg("child died")

				allProcs := db.GetProcs(core.WithAllIfNoFilters)

				procID, procFound := fun.FindKeyBy(allProcs, func(_ core.ProcID, procData core.Proc) bool {
					return procData.Status.Status == core.StatusRunning &&
						procData.Status.Pid == pid
				})
				if !procFound {
					continue
				}

				ebus.Publish(ctx, eventbus.NewPublishProcStopped(procID, status.ExitStatus(), eventbus.EmitReasonDied))
			}
		}
	}
}
