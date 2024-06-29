package cli

import (
	"strings"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/lo"
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
	db, _, errNewApp := app.New()
	if errNewApp != nil {
		log.Error().Err(errNewApp).Msg("new app")
		return nil, cobra.ShellCompDirectiveError
	}

	return iter.Map(listProcs(db).
		Filter(func(p core.ProcStat) bool {
			return strings.HasPrefix(p.Name, prefix)
		}).Seq,
		func(proc core.ProcStat) string {
			return proc.Name
			// Description: fun.Valid("status: " + proc.Status.String()),
		}).
		ToSlice(), cobra.ShellCompDirectiveNoFileComp
}

func completeFlagTag(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	db, _, errNewApp := app.New()
	if errNewApp != nil {
		log.Error().Err(errNewApp).Msg("new app")
		return nil, cobra.ShellCompDirectiveError
	}

	res := listProcs(db).
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
	db, _, errNewApp := app.New() // TODO: reduce number of calls to app.New
	if errNewApp != nil {
		log.Error().Err(errNewApp).Msg("new app")
		return nil, cobra.ShellCompDirectiveError
	}

	return iter.Map(listProcs(db).
		Filter(func(p core.ProcStat) bool {
			return strings.HasPrefix(string(p.ID), prefix)
		}).Seq,
		func(proc core.ProcStat) string {
			return proc.ID.String()
			// Description: fun.Valid("name: " + proc.Name),
		}).
		ToSlice(), cobra.ShellCompDirectiveNoFileComp
}

func completeArgGenericSelector(
	cmd *cobra.Command, args []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	names, _ := completeFlagName(cmd, args, prefix)
	tags, _ := completeFlagTag(cmd, args, prefix)
	return lo.Flatten(names, tags), cobra.ShellCompDirectiveNoFileComp
}
