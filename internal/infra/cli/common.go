package cli

import (
	"fmt"
	"strings"

	flags "github.com/rprtr258/cli/contrib"
	"github.com/rprtr258/fun/iter"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/log"
)

func printIDs(ids ...core.PMID) {
	for i, id := range ids {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(id)
	}
	fmt.Println()
}

type configFlag struct {
	Config *flags.Filename `short:"f" long:"config" description:"config file to use"`
}

type flagPMID core.PMID

func (f *flagPMID) Complete(match string) []flags.Completion {
	app, errNewApp := app.New()
	if errNewApp != nil {
		log.Error().Err(errNewApp).Msg("new app")
		return nil
	}

	return iter.Map(app.
		List().
		Filter(func(p core.Proc) bool {
			return strings.HasPrefix(string(p.ID), match)
		}),
		func(proc core.Proc) flags.Completion {
			return flags.Completion{
				Item:        proc.ID.String(),
				Description: "name: " + proc.Name,
			}
		}).
		ToSlice()
}

type flagProcName string

func (f *flagProcName) Complete(match string) []flags.Completion {
	app, errNewApp := app.New()
	if errNewApp != nil {
		log.Error().Err(errNewApp).Msg("new app")
		return nil
	}

	return iter.Map(app.
		List().
		Filter(func(p core.Proc) bool {
			return strings.HasPrefix(p.Name, match)
		}),
		func(proc core.Proc) flags.Completion {
			return flags.Completion{
				Item:        proc.Name,
				Description: "status: " + proc.Status.Status.String(),
			}
		}).
		ToSlice()
}

type flagProcTag string

func (f *flagProcTag) Complete(match string) []flags.Completion {
	app, errNewApp := app.New()
	if errNewApp != nil {
		log.Error().Err(errNewApp).Msg("new app")
		return nil
	}

	return iter.Map(iter.Unique(iter.FlatMap(app.
		List(),
		func(proc core.Proc) iter.Seq[string] {
			return iter.FromMany(proc.Tags...)
		}).
		Chain(iter.FromMany("all"))).
		Filter(func(tag string) bool {
			return strings.HasPrefix(tag, match)
		}),
		func(tag string) flags.Completion {
			return flags.Completion{
				Item:        tag,
				Description: "",
			}
		}).
		ToSlice()
}

type flagGenericSelector string

func (f *flagGenericSelector) Complete(match string) []flags.Completion {
	var fName flagProcName
	var fTag flagProcTag
	return lo.Flatten([][]flags.Completion{
		fName.Complete(match),
		fTag.Complete(match),
	})
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
