package cli

import (
	"fmt"
	"strings"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/lo"
)

func printProcs(procs ...core.Proc) {
	for _, proc := range procs {
		fmt.Println(proc.Name)
	}
}

func addFlagConfig(cmd *cobra.Command, config *string) {
	cmd.Flags().StringVarP(config, "config", "f", "", "config file to use")
}

func addFlagIDs(cmd *cobra.Command, ids *[]string) {
	cmd.Flags().StringSliceVar(ids, "id", nil, "id(s) of process(es) to list")
	cmd.RegisterFlagCompletionFunc("id", func(
		_ *cobra.Command, _ []string,
		prefix string,
	) ([]string, cobra.ShellCompDirective) {
		app, errNewApp := app.New()
		if errNewApp != nil {
			log.Error().Err(errNewApp).Msg("new app")
			return nil, cobra.ShellCompDirectiveError
		}

		return iter.Map(app.
			List().
			Filter(func(p core.Proc) bool {
				return strings.HasPrefix(string(p.ID), prefix)
			}),
			func(proc core.Proc) string {
				return proc.ID.String()
				// Description: fun.Valid("name: " + proc.Name),
			}).
			ToSlice(), cobra.ShellCompDirectiveNoFileComp
	})
}

func completeFlagName(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	app, errNewApp := app.New()
	if errNewApp != nil {
		log.Error().Err(errNewApp).Msg("new app")
		return nil, cobra.ShellCompDirectiveError
	}

	return iter.Map(app.
		List().
		Filter(func(p core.Proc) bool {
			return strings.HasPrefix(p.Name, prefix)
		}),
		func(proc core.Proc) string {
			return proc.Name
			// Description: fun.Valid("status: " + proc.Status.Status.String()),
		}).
		ToSlice(), cobra.ShellCompDirectiveNoFileComp
}

func addFlagNames(cmd *cobra.Command, names *[]string) {
	cmd.Flags().StringSliceVar(names, "name", nil, "name(s) of process(es) to list")
	cmd.RegisterFlagCompletionFunc("name", completeFlagName)
}

func completeFlagTag(
	_ *cobra.Command, _ []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	app, errNewApp := app.New()
	if errNewApp != nil {
		log.Error().Err(errNewApp).Msg("new app")
		return nil, cobra.ShellCompDirectiveError
	}

	// TODO: iter.Unique
	res := iter.FlatMap(app.
		List(),
		func(proc core.Proc) iter.Seq[string] {
			return iter.FromMany(proc.Tags...)
		}).
		Chain(iter.FromMany("all")).
		ToSlice()
	return fun.Filter[string](func(tag string) bool {
		return strings.HasPrefix(tag, prefix)
	}, fun.Uniq(res...)...), cobra.ShellCompDirectiveNoFileComp
}

func addFlagTags(cmd *cobra.Command, tags *[]string) {
	cmd.Flags().StringSliceVar(tags, "tag", nil, "tag(s) of process(es) to list")
	cmd.RegisterFlagCompletionFunc("tag", completeFlagTag)
}

func completeArgGenericSelector(
	cmd *cobra.Command, args []string,
	prefix string,
) ([]string, cobra.ShellCompDirective) {
	names, _ := completeFlagName(cmd, args, prefix)
	tags, _ := completeFlagTag(cmd, args, prefix)
	return lo.Flatten(names, tags), cobra.ShellCompDirectiveNoFileComp
}

// { Name: "link enable", PM2 I/O
// commander.command('link [secret] [public] [name]')
//   .option('--info-node [url]', 'set url info node')
//   .description('link with the pm2 monitoring dashboard')
// { Name: "link disble", commander.command('unlink')
//   .description('unlink with the pm2 monitoring dashboard')

// { Name: "monitor start",
// commander.command('monitor [name]')
//   .description('monitor target process / open monitoring dashboard')
// { Name: "monitor stop",
// commander.command('unmonitor [name]')
//   .description('unmonitor target process')
// { Name: "monit",
// Dashboard command
//   .alias('dash', "dashboard")
//   .description('launch termcaps monitoring')
//   .description('launch dashboard with monitoring and logs')
// { Name: "imonit",
//   .description('launch legacy termcaps monitoring')
//   .action(function() {
//     pm2.monit();
//   });
// },

// { Name: "send", commander.command('send <pm_id> <line>') .description('send stdin to <pm_id>')

// { Name: "attach", Attach to stdin/stdout
// commander.command('attach <pm_id> [command separator]')
//   .description('attach stdin/stdout to application identified by <pm_id>')

// { Name: "startup enable", commander.command('startup [platform]') .description('enable the pm2 startup hook')
// { Name: "startup disable", commander.command('unstartup') .description('disable the pm2 startup hook')

// instead dump process(es) into config of given format
// { Name: "ecosystem",
// Sample generate
// commander.command('ecosystem [mode]')
//   .alias('init')
//   .description('generate a process conf file. (mode = null or simple)')
//   .action(function(mode) {
//     pm2.generateSample(mode);
//   });
// },
// &cli.BoolFlag{Name:        "service-name", Usage: "define service name when generating startup script"},
// &cli.StringFlag{Name:      "home-path", Usage: "define home path when generating startup script"},
// &cli.StringFlag{Name:      "user", Aliases: []string{"u"}, Usage: "define user when generating startup script"},
// &cli.BoolFlag{Name:        "write", Aliases: []string{"w"}, Usage: "write configuration in local folder"},

// { Name:      "serve",
// 	Usage:     "serve a path over http",
// 	ArgsUsage: "[path] [port]",
// 	Aliases:   []string{"expose"},
// 	Flags:     []cli.Flag{
//   .option('--port [port]', 'specify port to listen to')
//   .option('--spa', 'always serving index.html on inexistent sub path')
//   .option('--basic-auth-username [username]', 'set basic auth username')
//   .option('--basic-auth-password [password]', 'set basic auth password')
