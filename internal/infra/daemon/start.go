package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
)

const CmdAgent = "agent"

var ErrAlreadyRunning = errors.New("process is already running")

// startAgent - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (app App) startAgent(id core.PMID) error {
	dbHandle := app.db
	errStart := func() error {
		pmExecutable, err := os.Executable()
		if err != nil {
			return xerr.NewWM(err, "get pm executable")
		}

		proc, ok := dbHandle.GetProc(id)
		if !ok {
			return xerr.NewM("not found proc to start", xerr.Fields{"pmid": id})
		}
		if proc.Status.Status == core.StatusRunning {
			return ErrAlreadyRunning
		}

		stdoutLogFile, err := os.OpenFile(proc.StdoutFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
		if err != nil {
			return xerr.NewWM(err, "open stdout file", xerr.Fields{"filename": proc.StdoutFile})
		}
		defer stdoutLogFile.Close()

		stderrLogFile, err := os.OpenFile(proc.StderrFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
		if err != nil {
			return xerr.NewWM(err, "open stderr file", xerr.Fields{"filename": proc.StderrFile})
		}
		defer func() {
			if errClose := stderrLogFile.Close(); errClose != nil {
				log.Error().Err(errClose).Send()
			}
		}()

		env := os.Environ()
		for k, v := range proc.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		env = append(env, fmt.Sprintf("%s=%s", _envPMID, proc.ID))

		procDesc, err := json.Marshal(proc)
		if err != nil {
			return xerr.NewWM(err, "marshal proc")
		}

		cmd := exec.Cmd{
			Path:   pmExecutable,
			Args:   []string{pmExecutable, CmdAgent, string(procDesc)},
			Dir:    proc.Cwd,
			Env:    env,
			Stdin:  os.Stdin,
			Stdout: stdoutLogFile,
			Stderr: stderrLogFile,
			SysProcAttr: &syscall.SysProcAttr{
				Setpgid: true,
			},
		}

		if err := cmd.Start(); err != nil {
			return xerr.NewWM(err, "running failed", xerr.Fields{"procData": procFields(proc)})
		}

		return nil
	}()

	if errStart != nil {
		if errStart != ErrAlreadyRunning {
			if errSetStatus := dbHandle.SetStatus(id, core.NewStatusInvalid()); errSetStatus != nil {
				log.Error().
					Err(errSetStatus).
					Stringer("pmid", id).
					Msg("failed to set proc status to invalid")
			}
			log.Error().
				Err(errStart).
				Stringer("pmid", id).
				Msg("failed to start proc")
		}
		log.Error().
			Stringer("pmid", id).
			Msg("already running")
	}

	dbHandle.StatusSetStarted(id)

	return nil
}

func deathCollector(ctx context.Context, db db.Handle) {
	// c := make(chan os.Signal, 10) // arbitrary buffer size
	// signal.Notify(c, syscall.SIGCHLD)

	// ticker := time.NewTicker(5 * time.Second)
	// defer ticker.Stop()

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		log.Info().Msg("context canceled, stopping...")
	// 		return
	// 	case <-ticker.C:
	// 		for procID, proc := range db.GetProcs(core.WithAllIfNoFilters) {
	// 			if proc.Status.Status != core.StatusRunning {
	// 				continue
	// 			}

	// 			process, ok := linuxprocess.StatPMID(proc.ID, "PM_PMID")
	// 			pid := 0
	// 			if ok {
	// 				pid = process.Pid
	// 			}

	// 			switch _, errStat := linuxprocess.ReadProcessStat(pid); errStat {
	// 			case nil:
	// 				// process stat file exists hence process is still running
	// 				continue
	// 			case linuxprocess.ErrStatFileNotFound:
	// 				log.Info().
	// 					Stringer("pid", proc.ID).
	// 					Msg("process seems to be stopped, updating status...")

	// 				db.StatusSetStopped(procID)
	// 				ebus.Publish(ctx, eventbus.NewPublishProcStopped(procID, eventbus.EmitReasonDied))
	// 			default:
	// 				log.Warn().
	// 					Err(errStat).
	// 					Stringer("pmid", proc.ID).
	// 					Msg("read proc stat")
	// 			}
	// 		}
	// 	case <-c:
	// 		// wait for any of childs' death
	// 		// TODO: get back/remove
	// 		for {
	// 			var status syscall.WaitStatus
	// 			pid, errWait := syscall.Wait4(-1, &status, 0, nil)
	// 			if pid < 0 {
	// 				break
	// 			}
	// 			if errWait != nil {
	// 				log.Error().Err(errWait).Msg("Wait4 failed")
	// 				continue
	// 			}

	// 			log.Info().Int("pid", pid).Msg("child died")

	// 			allProcs := db.GetProcs(core.WithAllIfNoFilters)

	// 			procID, procFound := fun.FindKeyBy(allProcs, func(_ core.PMID, procData core.Proc) bool {
	// 				return procData.Status.Status == core.StatusRunning &&
	// 					procData.ID == pid
	// 			})
	// 			if !procFound {
	// 				continue
	// 			}

	// 			daemon.StatusSetStopped(db, procID)
	// 			ebus.Publish(ctx, eventbus.NewPublishProcStopped(procID, eventbus.EmitReasonDied))
	// 		}
	// 	}
	// }
}

func (app App) StartRaw(proc core.Proc) error {
	stdoutLogFile, err := os.OpenFile(proc.StdoutFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return xerr.NewWM(err, "open stdout file", xerr.Fields{"filename": proc.StdoutFile})
	}
	defer stdoutLogFile.Close()

	stderrLogFile, err := os.OpenFile(proc.StderrFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return xerr.NewWM(err, "open stderr file", xerr.Fields{"filename": proc.StderrFile})
	}
	defer func() {
		if errClose := stderrLogFile.Close(); errClose != nil {
			log.Error().Err(errClose).Send()
		}
	}()

	env := os.Environ()
	for k, v := range proc.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Cmd{
		Path:   proc.Command,
		Args:   append([]string{proc.Command}, proc.Args...),
		Dir:    proc.Cwd,
		Env:    env,
		Stdin:  os.Stdin,
		Stdout: stdoutLogFile,
		Stderr: stderrLogFile,
		SysProcAttr: &syscall.SysProcAttr{
			Setpgid: true,
		},
	}

	if err = cmd.Start(); err != nil {
		app.db.StatusSetStopped(proc.ID)
		return xerr.NewWM(err, "running failed", xerr.Fields{"procData": proc})
	}

	app.db.StatusSetStarted(proc.ID)

	err = cmd.Wait()
	// TODO: use status code to update stopped
	app.db.StatusSetStopped(proc.ID)
	return err
}

// Start already created processes
func (app App) Start(ids ...core.PMID) error {
	for _, id := range ids {
		if errStart := app.startAgent(id); errStart != nil {
			return xerr.NewWM(errStart, "start processes")
		}
	}

	return nil
}
