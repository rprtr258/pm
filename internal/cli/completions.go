package cli

import (
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

func completeArgGenericSelector(filter filterType) func(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	return func(
		_ *cobra.Command, _ []string,
		prefix string,
	) ([]string, cobra.ShellCompDirective) {
		names, _ := completeFlagName(filter)(prefix)
		tags, _ := completeFlagTag(filter)(prefix)
		ids, _ := completeFlagIDs(filter)(prefix)

		flatten := make([]string, 0, len(names)+len(tags)+len(ids))
		flatten = append(flatten, names...)
		flatten = append(flatten, tags...)
		flatten = append(flatten, ids...)

		return flatten, cobra.ShellCompDirectiveNoFileComp
	}
}
