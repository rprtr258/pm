package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func registerFlagCompletionFunc(
	c *cobra.Command,
	name string,
	f func(toComplete string) ([]string, cobra.ShellCompDirective),
) {
	if err := c.RegisterFlagCompletionFunc(
		name,
		func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return f(toComplete)
		}); err != nil {
		log.Panic().
			Err(err).
			Str("flagName", name).
			Str("command", c.Name()).
			Msg("failed to register flag completion func")
	}
}

func completeFlagName(prefix string) ([]string, cobra.ShellCompDirective) {
	return slices.Collect(func(yield func(string) bool) {
		for proc := range listProcs(dbb).Seq {
			if strings.HasPrefix(proc.Name, prefix) && !yield(fmt.Sprintf("%s\tproc: %s", proc.Name, proc.Status.String())) {
				break
			}
		}
	}), cobra.ShellCompDirectiveNoFileComp
}

func completeFlagTag(prefix string) ([]string, cobra.ShellCompDirective) {
	return slices.Collect(func(yield func(string) bool) {
		for tag := range listProcs(dbb).Tags() {
			if strings.HasPrefix(tag, prefix) && !yield(tag) {
				break
			}
		}
	}), cobra.ShellCompDirectiveNoFileComp
}

func completeFlagIDs(prefix string) ([]string, cobra.ShellCompDirective) {
	return slices.Collect(func(yield func(string) bool) {
		for proc := range listProcs(dbb).Seq {
			if strings.HasPrefix(string(proc.ID), prefix) && !yield(fmt.Sprintf("%s\tname: %s", proc.ID.String(), proc.Name)) {
				break
			}
		}
	}), cobra.ShellCompDirectiveNoFileComp
}

func completeArgGenericSelector( // TODO: add and use isrunning flag
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	names, _ := completeFlagName(prefix)
	tags, _ := completeFlagTag(prefix)

	flatten := make([]string, 0, len(names)+len(tags))
	flatten = append(flatten, names...)
	flatten = append(flatten, tags...)

	return flatten, cobra.ShellCompDirectiveNoFileComp
}
