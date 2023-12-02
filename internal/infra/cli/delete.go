package cli

import (
	"fmt"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type _cmdDelete struct {
	Names []flagProcName `long:"name" description:"name(s) of process(es) to list"`
	Tags  []flagProcTag  `long:"tag" description:"tag(s) of process(es) to list"`
	IDs   []flagPMID     `long:"id" description:"id(s) of process(es) to list"`
	Args  struct {
		Rest []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
	configFlag
}

func (x *_cmdDelete) Execute(_ []string) error {
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
			fmt.Println("Nothing to delete, leaving")
			return nil
		}

		if err := app.Stop(procIDs...); err != nil {
			return xerr.NewWM(err, "delete")
		}

		if errDelete := app.Delete(procIDs...); errDelete != nil {
			return xerr.NewWM(errDelete, "delete")
		}

		return nil
	}

	configs, errLoadConfigs := core.LoadConfigs(string(*x.configFlag.Config))
	if errLoadConfigs != nil {
		return xerr.NewWM(errLoadConfigs, "load configs", xerr.Fields{
			"config": string(*x.configFlag.Config),
		})
	}

	list, errList := app.ListByRunConfigs(configs)
	if errList != nil {
		return xerr.NewWM(errList, "list by run configs", xerr.Fields{"configs": configs})
	}

	procIDs := core.FilterProcMap(
		list,
		core.WithGeneric(args...),
		core.WithIDs(ids...),
		core.WithNames(names...),
		core.WithTags(tags...),
		core.WithAllIfNoFilters,
	)

	if err := app.Stop(procIDs...); err != nil {
		return xerr.NewWM(err, "stop")
	}

	if err := app.Delete(procIDs...); err != nil {
		return xerr.NewWM(err, "delete")
	}

	return nil
}

func deferErr(closer func() error) func() {
	return func() {
		if err := closer(); err != nil {
			log.Error().Err(err).Msg("some defer action failed:")
		}
	}
}
