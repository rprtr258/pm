package cli

import (
	"context"
	"fmt"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type _cmdSignal struct {
	// &cli.BoolFlag{
	// 	Name:  "kill",
	// 	Usage: "kill process with SIGKILL instead of SIGINT",
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
		Signal string                `positional-arg-name:"signal"`
		Rest   []flagGenericSelector `positional-arg-name:"name|tag|id"`
	} `positional-args:"yes"`
	configFlag
}

func (x _cmdSignal) Execute(ctx context.Context) error {
	var sig syscall.Signal
	switch x.Args.Signal {
	case "SIGKILL":
		sig = syscall.SIGKILL
	case "SIGTERM":
		sig = syscall.SIGTERM
	case "SIGINT":
		sig = syscall.SIGINT
	default:
		return xerr.NewF("unknown signal", map[string]any{"signal": x.Args.Signal})
	}

	client, errList := app.New()
	if errList != nil {
		return errors.Wrap(errList, "new grpc client")
	}

	list := client.List()

	if x.configFlag.Config != nil {
		configs, errLoadConfigs := core.LoadConfigs(string(*x.configFlag.Config))
		if errLoadConfigs != nil {
			return errors.Wrap(errLoadConfigs, "load configs")
		}

		names := fun.FilterMap[string](func(cfg core.RunConfig) fun.Option[string] {
			return cfg.Name
		}, configs...)

		list = list.
			Filter(func(proc core.Proc) bool {
				return fun.Contains(proc.Name, names...)
			})
	}

	procIDs := iter.Map(list.
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
		fmt.Println("nothing to stop")
		return nil
	}

	if err := client.Signal(sig, procIDs...); err != nil {
		return errors.Wrap(err, "client.stop")
	}

	return nil
}
