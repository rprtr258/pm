package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/config"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/db"
)

var dbb, cfg = func() (db.Handle, core.Config) {
	db, config, errNewApp := config.New()
	if errNewApp != nil {
		log.Panic().Err(errNewApp).Msg("new app")
	}
	return db, config
}()
var seq = listProcs(dbb)

func printProcs(procs ...core.ProcStat) {
	for _, proc := range procs {
		fmt.Println(proc.Name)
	}
}

func addFlagConfig(cmd *cobra.Command, config *string) {
	cmd.Flags().StringVarP(config, "config", "f", "", "config file to use")
}

func addFlagStrings(
	cmd *cobra.Command,
	dest *[]string,
	long string,
	description string,
	completeFunc func(string) ([]string, cobra.ShellCompDirective),
) {
	cmd.Flags().StringSliceVar(dest, long, nil, description)
	registerFlagCompletionFunc(cmd, long, completeFunc)
}

func completeFlagName(filter filterType) func(prefix string) ([]string, cobra.ShellCompDirective) {
	return func(prefix string) ([]string, cobra.ShellCompDirective) {
		return slices.Collect(func(yield func(string) bool) {
			for proc := range seq.FilterRunning(filter).Seq {
				if strings.HasPrefix(proc.Name, prefix) && !yield(fmt.Sprintf("%s\tproc: %s", proc.Name, proc.Status.String())) {
					break
				}
			}
		}), cobra.ShellCompDirectiveNoFileComp
	}
}

func completeFlagTag(filter filterType) func(prefix string) ([]string, cobra.ShellCompDirective) {
	return func(prefix string) ([]string, cobra.ShellCompDirective) {
		return slices.Collect(func(yield func(string) bool) {
			for tag := range seq.FilterRunning(filter).Tags() {
				if strings.HasPrefix(tag, prefix) && !yield(tag) {
					break
				}
			}
		}), cobra.ShellCompDirectiveNoFileComp
	}
}

func completeFlagIDs(filter filterType) func(prefix string) ([]string, cobra.ShellCompDirective) {
	return func(prefix string) ([]string, cobra.ShellCompDirective) {
		return slices.Collect(func(yield func(string) bool) {
			for proc := range seq.FilterRunning(filter).Seq {
				if strings.HasPrefix(string(proc.ID), prefix) && !yield(fmt.Sprintf("%s\tname: %s", proc.ID.String(), proc.Name)) {
					break
				}
			}
		}), cobra.ShellCompDirectiveNoFileComp
	}
}

func addFlagGenerics(
	cmd *cobra.Command,
	filter filterType,
	names, tags, ids *[]string,
) {
	addFlagStrings(cmd, names, "name", "name(s) of process(es)", completeFlagName(filter))
	addFlagStrings(cmd, tags, "tag", "tag(s) of process(es)", completeFlagTag(filter))
	addFlagStrings(cmd, ids, "id", "id(s) of process(es) to list", completeFlagIDs(filter))
}

func addFlagInteractive(cmd *cobra.Command, dest *bool) {
	cmd.Flags().BoolVarP(dest, "interactive", "i", false, "prompt before taking action")
}

func confirmProc(ps core.ProcStat, action string) bool {
	var result bool
	if err := huh.NewConfirm().
		Title(fmt.Sprintf(
			"Do you really want to %s process %q id=%s ? ",
			action, ps.Name, ps.ID.String(),
		)).
		Inline(true).
		Value(&result).
		WithTheme(theme()).
		Run(); err != nil {
		log.Panic().Msg(err.Error())
	}
	return result
}
