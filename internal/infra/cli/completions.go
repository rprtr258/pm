package cli

import (
	"strings"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/db"
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

type completer struct{ db db.Handle }

var compl = completer{
	db: func() db.Handle {
		db, _, errNewApp := app.New() // TODO: call app.New only once
		if errNewApp != nil {
			log.Fatal().Err(errNewApp).Msg("new app")
		}
		return db
	}(),
}

func (c completer) FlagName(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	return iter.Map(listProcs(c.db).
		Filter(func(p core.ProcStat) bool {
			return strings.HasPrefix(p.Name, prefix)
		}).Seq,
		func(proc core.ProcStat) string {
			return proc.Name
			// Description: fun.Valid("status: " + proc.Status.String()),
		}).
		ToSlice(), cobra.ShellCompDirectiveNoFileComp
}

func (c completer) FlagTag(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	res := listProcs(c.db).
		Tags().
		ToSlice()
	return fun.Filter(func(tag string) bool {
		return strings.HasPrefix(tag, prefix)
	}, res...), cobra.ShellCompDirectiveNoFileComp
}

func (c completer) FlagIDs(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	return iter.Map(listProcs(c.db).
		Filter(func(p core.ProcStat) bool {
			return strings.HasPrefix(string(p.ID), prefix)
		}).Seq,
		func(proc core.ProcStat) string {
			return proc.ID.String()
			// Description: fun.Valid("name: " + proc.Name),
		}).
		ToSlice(), cobra.ShellCompDirectiveNoFileComp
}

func (c completer) ArgGenericSelector(
	cmd *cobra.Command, args []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	names, _ := c.FlagName(cmd, args, prefix)
	tags, _ := c.FlagTag(cmd, args, prefix)
	return lo.Flatten(names, tags), cobra.ShellCompDirectiveNoFileComp
}
