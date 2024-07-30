package cli

import (
	stdErrors "errors"
	"fmt"
	"os"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func implSignal(
	sig syscall.Signal,
	ids ...core.PMID,
) error {
	list := linuxprocess.List()
	return errors.Combine(fun.Map[error](func(id core.PMID) error {
		return errors.Wrapf(func() error {
			osProc, ok := linuxprocess.StatPMID(list, id)
			if !ok {
				return errors.Newf("get process by pmid, id=%s signal=%s", id, sig.String())
			}

			if errKill := syscall.Kill(-osProc.ShimPID, sig); errKill != nil {
				switch {
				case stdErrors.Is(errKill, os.ErrProcessDone):
					return errors.New("tried to send signal to process which is done")
				case stdErrors.Is(errKill, syscall.ESRCH): // no such process
					return errors.New("tried to send signal to process which doesn't exist")
				default:
					return errors.Wrapf(errKill, "kill process, pid=%d", osProc.ShimPID)
				}
			}

			return nil
		}(), "pmid=%s", id)
	}, ids...)...)
}

var _cmdSignal = func() *cobra.Command {
	var names, ids, tags []string
	var config string
	var interactive bool
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

			list := listProcs(dbb)

			if config != nil {
				configs, errLoadConfigs := core.LoadConfigs(*config)
				if errLoadConfigs != nil {
					return errors.Wrap(errLoadConfigs, "load configs")
				}

				namesFilter := fun.Map[string](func(cfg core.RunConfig) string {
					return cfg.Name
				}, configs...)

				list = list.
					Filter(func(proc core.ProcStat) bool {
						return fun.Contains(proc.Name, namesFilter...)
					})
			}

			filterFunc := core.FilterFunc(
				core.WithGeneric(args...),
				core.WithIDs(ids...),
				core.WithNames(names...),
				core.WithTags(tags...),
				core.WithAllIfNoFilters,
			)
			procs := list.
				Filter(func(ps core.ProcStat) bool {
					return ps.Status != core.StatusStopped &&
						filterFunc(ps.Proc) &&
						(!interactive || confirmProc(ps, "signal"))
				}).
				ToSlice()
			if len(procs) == 0 {
				fmt.Println("nothing to stop")
				return nil
			}

			procIDs := fun.Map[core.PMID](func(proc core.ProcStat) core.PMID { return proc.ID }, procs...)
			if err := implSignal(sig, procIDs...); err != nil {
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
	addFlagInteractive(cmd, &interactive)
	addFlagNames(cmd, &names)
	addFlagTags(cmd, &tags)
	addFlagIDs(cmd, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
