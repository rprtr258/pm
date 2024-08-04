package cli

import (
	"fmt"
	"strings"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
)

func registerFlagCompletionFunc(
	c *cobra.Command,
	name string,
	f func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective),
) {
	if err := c.RegisterFlagCompletionFunc(name, f); err != nil {
		log.Fatal().
			Err(err).
			Str("flagName", name).
			Str("command", c.Name()).
			Msg("failed to register flag completion func")
	}
}

func completeFlagName(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	return iter.Map(listProcs(dbb).
		Filter(func(p core.ProcStat) bool {
			return strings.HasPrefix(p.Name, prefix)
		}).Seq,
		func(proc core.ProcStat) string {
			return fmt.Sprintf("%s\tproc: %s", proc.Name, proc.Status.String())
		}).
		ToSlice(), cobra.ShellCompDirectiveNoFileComp
}

func completeFlagTag(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	res := listProcs(dbb).
		Tags().
		ToSlice()
	return fun.Filter(func(tag string) bool {
		return strings.HasPrefix(tag, prefix)
	}, res...), cobra.ShellCompDirectiveNoFileComp
}

func completeFlagIDs(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	return iter.Map(listProcs(dbb).
		Filter(func(p core.ProcStat) bool {
			return strings.HasPrefix(string(p.ID), prefix)
		}).Seq,
		func(proc core.ProcStat) string {
			return fmt.Sprintf("%s\tname: %s", proc.ID.String(), proc.Name)
		}).
		ToSlice(), cobra.ShellCompDirectiveNoFileComp
}

func completeArgGenericSelector(
	cmd *cobra.Command,
	args []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	names, _ := completeFlagName(cmd, args, prefix)
	tags, _ := completeFlagTag(cmd, args, prefix)

	flatten := make([]string, 0, len(names)+len(tags))
	flatten = append(flatten, names...)
	flatten = append(flatten, tags...)

	return flatten, cobra.ShellCompDirectiveNoFileComp
}
