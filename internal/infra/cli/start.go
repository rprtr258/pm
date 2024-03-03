package cli

import (
	"context"
	"fmt"

	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/pm/internal/infra/errors"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type _cmdStart struct {
	Names []flagProcName `long:"name" description:"name(s) of process(es) to list"`
	Tags  []flagProcTag  `long:"tag" description:"tag(s) of process(es) to list"`
	IDs   []flagPMID     `long:"id" description:"id(s) of process(es) to list"`
	Args  struct {
		Rest []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
	configFlag
}

func (x _cmdStart) Execute(ctx context.Context) error {
	app, errNewApp := app.New()
	if errNewApp != nil {
		return errors.Wrap(errNewApp, "new app")
	}

	list := app.List()

	if x.configFlag.Config == nil {
		procIDs := iter.Map(list.
			Filter(core.FilterFunc(
				core.WithGeneric(x.Args.Rest...),
				core.WithIDs(x.IDs...),
				core.WithNames(x.Names...),
				core.WithTags(x.Tags...),
			)),
			func(proc core.Proc) core.PMID {
				return proc.ID
			}).
			ToSlice()
		if len(procIDs) == 0 {
			fmt.Println("nothing to start")
			return nil
		}

		if err := app.Start(procIDs...); err != nil {
			return errors.Wrap(err, "client.start")
		}

		printIDs(procIDs...)

		return nil
	}

	configs, errLoadConfigs := core.LoadConfigs(string(*x.configFlag.Config))
	if errLoadConfigs != nil {
		return errors.Wrap(errLoadConfigs, "load configs: %s", string(*x.configFlag.Config))
	}

	procIDs := iter.Map(app.
		ListByRunConfigs(configs).
		Filter(core.FilterFunc(
			core.WithGeneric(x.Args.Rest...),
			core.WithIDs(x.IDs...),
			core.WithNames(x.Names...),
			core.WithTags(x.Tags...),
			core.WithAllIfNoFilters,
		)),
		func(proc core.Proc) core.PMID {
			return proc.ID
		}).
		ToSlice()
	if len(procIDs) == 0 {
		fmt.Println("nothing to start")
		return nil
	}

	if err := app.Start(procIDs...); err != nil {
		return errors.Wrap(err, "client.start")
	}

	printIDs(procIDs...)

	return nil
}
