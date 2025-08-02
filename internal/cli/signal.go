package cli

import (
	stdErrors "errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/errors"
	"github.com/rprtr258/pm/internal/linuxprocess"
)

func implSignal(
	sig syscall.Signal,
	ids ...core.PMID,
) error {
	list := linuxprocess.List()

	pidsToSignal := map[int]core.PMID{}
	for _, id := range ids {
		stat, ok := linuxprocess.StatPMID(list, id)
		if !ok {
			return errors.Newf("get process by pmid, id=%s signal=%s", id, sig.String())
		}

		if stat.ChildPID == 0 {
			continue
		}

		pidsToSignal[stat.ChildPID] = id
		for _, child := range linuxprocess.Children(list, stat.ChildPID) {
			pidsToSignal[child.Handle.Pid] = id
		}
	}

	errs := []error{}
	for pid, pmid := range pidsToSignal {
		if err := func() error {
			if errKill := syscall.Kill(-pid, sig); errKill != nil {
				switch {
				case stdErrors.Is(errKill, os.ErrProcessDone):
					return errors.New("tried to send signal to process which is done")
				case stdErrors.Is(errKill, syscall.ESRCH): // no such process
					return errors.New("tried to send signal to process which doesn't exist")
				default:
					return errors.Wrapf(errKill, "kill process")
				}
			}

			return nil
		}(); err != nil {
			errs = append(errs, errors.Wrapf(err, "pmid=%s, pid=%d", pmid, pid))
		}
	}
	return errors.Combine(errs...)
}

var (
	_signals = func() map[string]syscall.Signal {
		names := map[string]syscall.Signal{
			"SIGHUP":  syscall.SIGHUP,
			"SIGINT":  syscall.SIGINT,
			"SIGQUIT": syscall.SIGQUIT,
			"SIGKILL": syscall.SIGKILL,
			"SIGTERM": syscall.SIGTERM,
			"SIGCONT": syscall.SIGCONT,
			"SIGSTOP": syscall.SIGSTOP,
		}

		res := make(map[string]syscall.Signal, 2*len(names))
		for name, sig := range names {
			res[name] = sig
			res[strings.TrimPrefix(name, "SIG")] = sig
		}
		return res
	}()
	_signalNames = func() []string {
		res := fun.Keys(_signals)
		sort.Strings(res)
		return res
	}()
)

var _cmdSignal = func() *cobra.Command {
	const filter = filterRunning
	var names, ids, tags []string
	var config string
	var interactive bool
	cmd := &cobra.Command{
		Use:     "signal [sigspec] [name|tag|id]...",
		Short:   "send signal to process(es)",
		Aliases: []string{"kill"},
		ValidArgsFunction: func(
			cmd *cobra.Command,
			args []string,
			prefix string,
		) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return _signalNames, cobra.ShellCompDirectiveNoFileComp
			}

			return completeArgGenericSelector(filter)(cmd, args, prefix)
		},
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			signal := args[0]
			args = args[1:]
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)

			sig, ok := _signals[signal]
			if !ok {
				if x, err := strconv.Atoi(signal); err == nil {
					sig = syscall.Signal(x)
				} else {
					return errors.Newf("unknown signal: %q", signal)
				}
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
				Slice()
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
	addFlagInteractive(cmd, &interactive)
	addFlagGenerics(cmd, filter, &names, &tags, &ids)
	addFlagConfig(cmd, &config)
	return cmd
}()
