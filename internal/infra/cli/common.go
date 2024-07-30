package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/db"
)

var dbb, cfg = func() (db.Handle, core.Config) {
	db, config, errNewApp := app.New()
	if errNewApp != nil {
		log.Fatal().Err(errNewApp).Msg("new app")
	}
	return db, config
}()

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
	completeFunc func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective),
) {
	cmd.Flags().StringSliceVar(dest, long, nil, description)
	registerFlagCompletionFunc(cmd, long, completeFunc)
}

func addFlagNames(cmd *cobra.Command, names *[]string) {
	addFlagStrings(cmd, names, "name", "name(s) of process(es)", completeFlagName)
}

func addFlagTags(cmd *cobra.Command, tags *[]string) {
	addFlagStrings(cmd, tags, "tag", "tag(s) of process(es)", completeFlagTag)
}

func addFlagIDs(cmd *cobra.Command, ids *[]string) {
	addFlagStrings(cmd, ids, "id", "id(s) of process(es) to list", completeFlagIDs)
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
		log.Fatal().Msg(err.Error())
	}
	return result
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
