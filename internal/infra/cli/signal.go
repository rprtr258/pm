package cli

import (
	stdErrors "errors"
	"fmt"
	"os"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func implSignal(
	appp app.App,
	sig syscall.Signal,
	ids ...core.PMID,
) error {
	var merr error
	for _, id := range ids {
		// signal - send signal to process
		if err := func() error {
			proc, ok := appp.DB.GetProc(id)
			if !ok {
				return errors.New("not found proc to stop")
			}

			if proc.Status.Status != core.StatusRunning {
				return errors.New("proc is not running, can't send signal")
			}

			osProc, ok := linuxprocess.StatPMID(id, app.EnvPMID)
			if !ok {
				return errors.Newf("get process by pmid, id=%s signal=%s", id, sig.String())
			}

			if errKill := syscall.Kill(-osProc.Pid, sig); errKill != nil {
				switch {
				case stdErrors.Is(errKill, os.ErrProcessDone):
					return errors.New("tried to send signal to process which is done")
				case stdErrors.Is(errKill, syscall.ESRCH): // no such process
					return errors.New("tried to send signal to process which doesn't exist")
				default:
					return errors.Wrapf(errKill, "kill process, pid=%d", osProc.Pid)
				}
			}

			return nil
		}(); err != nil {
			multierr.AppendInto(&merr, errors.Wrapf(err, "pmid=%s", id))
		}
	}
	return merr
}

var _cmdSignal = func() *cobra.Command {
	var names, ids, tags []string
	var config string
	cmd := &cobra.Command{
		Use:               "signal SIGNAL [name|tag|id]...",
		Short:             "send signal to process(es)",
		Aliases:           []string{"kill"},
		GroupID:           "inspection",
		ValidArgsFunction: completeArgGenericSelector,
		Args:              cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			signal := args[0]
			args = args[1:]
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			var sig syscall.Signal
			switch signal {
			case "SIGKILL", "9":
				sig = syscall.SIGKILL
			case "SIGTERM", "15":
				sig = syscall.SIGTERM
			case "SIGINT", "2":
				sig = syscall.SIGINT
			default:
				return errors.Newf("unknown signal: %q", signal)
			}

			appp, errList := app.New()
			if errList != nil {
				return errors.Wrap(errList, "new grpc client")
			}

			list := appp.List()

			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrap(errLoadConfigs, "load configs")
				}

				names := fun.FilterMap[string](func(cfg core.RunConfig) fun.Option[string] {
					return cfg.Name
				}, configs...)

				list = list.
					Filter(func(proc core.Proc) bool {
						return fun.Contains(proc.Name, names...)
					})
			}

			procIDs := iter.Map(list.
				Filter(core.FilterFunc(
					core.WithGeneric(args...),
					core.WithIDs(ids...),
					core.WithNames(names...),
					core.WithTags(tags...),
					core.WithAllIfNoFilters,
				)),
				func(proc core.Proc) core.PMID {
					return proc.ID
				}).
				ToSlice()
			if len(procIDs) == 0 {
				fmt.Println("nothing to stop")
				return nil
			}

			if err := implSignal(appp, sig, procIDs...); err != nil {
				return errors.Wrapf(err, "client.stop signal=%v", sig)
			}

			return nil
		},
	}
	// &cli.BoolFlag{
	// 	Name:  "kill",
	// 	Usage: "kill process with SIGKILL instead of SIGINT",
	// },
	// &cli.BoolFlag{
	// 	Name:  "no-treekill",
	// 	Usage: "Only kill the main process, not detached children",
	// },
	// TODO: -i/... to confirm which procs will be stopped
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
