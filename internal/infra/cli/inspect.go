package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/daemon"
)

var _cmdInspect = &cli.Command{
	Name:     "inspect",
	Aliases:  []string{"i"},
	Usage:    "inspect processes",
	Category: "inspection",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: "name(s) of process(es) to list",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "tag(s) of process(es) to list",
		},
		&cli.StringSliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to list",
		},
	},
	Action: func(ctx *cli.Context) error {
		return inspect(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			fun.Map[core.PMID](ctx.StringSlice("id"), func(id string) core.PMID {
				return core.PMID(id)
			}),
		)
	},
}

func inspect(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []core.PMID,
) error {
	app, errNewApp := daemon.New()
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	list := app.List() // TODO: move in filters which are bit below

	procIDsToShow := core.FilterProcMap(
		list,
		core.NewFilter(
			core.WithAllIfNoFilters,
			core.WithGeneric(genericFilters),
			core.WithIDs(idFilters...),
			core.WithNames(nameFilters),
			core.WithTags(tagFilters),
		),
	)

	procsToShow := fun.MapDict(procIDsToShow, list)
	for _, proc := range procsToShow {
		b, _ := json.Marshal(proc)
		fmt.Println(string(b))
	}

	return nil
}
