package cli

import (
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type _cmdStop struct {
	// &cli.BoolFlag{
	// 	Name:  "watch",
	// 	Usage: "stop watching for file changes",
	// },
	// &cli.BoolFlag{
	// 	Name:  "kill",
	// 	Usage: "kill process with SIGKILL instead of SIGINT",
	// },
	// &cli.DurationFlag{
	// 	Name:    "kill-timeout",
	// 	Aliases: []string{"k"},
	// 	Usage:   "delay before sending final SIGKILL signal to process",
	// },
	// &cli.BoolFlag{
	// 	Name:  "no-treekill",
	// 	Usage: "Only kill the main process, not detached children",
	// },
	// TODO: -i/... to confirm which procs will be stopped
	Names []flagProcName `long:"name" description:"name(s) of process(es) to list"`
	Tags  []flagProcTag  `long:"tag" description:"tag(s) of process(es) to list"`
	IDs   []flagPMID     `long:"id" description:"id(s) of process(es) to list"`
	Args  struct {
		Rest []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
	configFlag
}

func (x *_cmdStop) Execute(_ []string) error {
	client, errList := app.New()
	if errList != nil {
		return xerr.NewWM(errList, "new grpc client")
	}

	list := client.List()

	if x.configFlag.Config == nil {
		configs, errLoadConfigs := core.LoadConfigs(string(*x.configFlag.Config))
		if errLoadConfigs != nil {
			return xerr.NewWM(errLoadConfigs, "load configs")
		}

		names := fun.FilterMap[string](func(cfg core.RunConfig) fun.Option[string] {
			return cfg.Name
		}, configs...)

		configList := lo.PickBy(list, func(_ core.PMID, procData core.Proc) bool {
			return fun.Contains(procData.Name, names...)
		})
		list = configList
	}

	procIDs := core.FilterProcMap(
		list,
		core.WithGeneric(x.Args.Rest...),
		core.WithIDs(x.IDs...),
		core.WithNames(x.Names...),
		core.WithTags(x.Tags...),
		core.WithAllIfNoFilters,
	)
	if len(procIDs) == 0 {
		fmt.Println("nothing to stop")
		return nil
	}

	if err := client.Stop(procIDs...); err != nil {
		return xerr.NewWM(err, "client.stop")
	}

	return nil
}
