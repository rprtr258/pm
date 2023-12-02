package cli

import (
	"fmt"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type _cmdRestart struct {
	Names []flagProcName `long:"name" description:"name(s) of process(es) to list"`
	Tags  []flagProcTag  `long:"tag" description:"tag(s) of process(es) to list"`
	IDs   []flagPMID     `long:"id" description:"id(s) of process(es) to list"`
	Args  struct {
		Rest []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
	configFlag
}

func (x *_cmdRestart) Execute(_ []string) error {
	names := x.Names
	tags := x.Tags
	ids := x.IDs
	args := x.Args.Rest

	app, errNewApp := app.New()
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	list := app.List()

	if x.configFlag.Config == nil {
		procIDs := core.FilterProcMap(
			list,
			core.WithGeneric(args...),
			core.WithIDs(ids...),
			core.WithNames(names...),
			core.WithTags(tags...),
		)

		if len(procIDs) == 0 {
			fmt.Println("nothing to restart")
			return nil
		}

		if err := app.Stop(procIDs...); err != nil {
			return xerr.NewWM(err, "client.stop")
		}

		if errStart := app.Start(procIDs...); errStart != nil {
			return xerr.NewWM(errStart, "client.start")
		}

		return nil
	}

	configFile := string(*x.configFlag.Config)

	configs, errLoadConfigs := core.LoadConfigs(configFile)
	if errLoadConfigs != nil {
		return xerr.NewWM(errLoadConfigs, "load configs", xerr.Fields{
			"config": configFile,
		})
	}

	filteredList, err := app.ListByRunConfigs(configs)
	if err != nil {
		return xerr.NewWM(err, "list procs by configs")
	}

	// TODO: reuse filter options
	procIDs := core.FilterProcMap(
		filteredList,
		core.WithGeneric(args...),
		core.WithIDs(ids...),
		core.WithNames(names...),
		core.WithTags(tags...),
		core.WithAllIfNoFilters,
	)

	if len(procIDs) == 0 {
		fmt.Println("nothing to start")
		return nil
	}

	if errStop := app.Stop(procIDs...); errStop != nil {
		return xerr.NewWM(errStop, "client.stop")
	}

	if errStart := app.Start(procIDs...); errStart != nil {
		return xerr.NewWM(err, "client.start")
	}

	return nil
}
