package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/db"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
)

func init() {
	AllCmds = append(AllCmds, DeleteCmd)
}

var DeleteCmd = &cli.Command{
	Name:      "delete",
	Aliases:   []string{"del", "rm"},
	Usage:     "stop and remove process(es)",
	ArgsUsage: "<name|id|namespace|tag|json>...",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: "name(s) of process(es) to stop and remove",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "tag(s) of process(es) to stop and remove",
		},
		&cli.Uint64SliceFlag{
			Name:  "id",
			Usage: "id(s) of process(es) to stop and remove",
		},
	},
	Action: func(ctx *cli.Context) error {
		return delete(
			ctx.Context,
			ctx.Args().Slice(),
			ctx.StringSlice("name"),
			ctx.StringSlice("tags"),
			ctx.Uint64Slice("id"),
		)
	},
}

func delete(
	ctx context.Context,
	genericFilters, nameFilters, tagFilters []string,
	idFilters []uint64,
) error {
	client, deferFunc, err := NewGrpcClient()
	if err != nil {
		return err
	}
	defer deferErr(deferFunc)

	dbHandle := db.New(_daemonDBFile)

	resp, err := dbHandle.List()
	if err != nil {
		return err
	}

	procIDs := internal.FilterProcs[uint64](
		resp,
		internal.WithGeneric(genericFilters),
		internal.WithIDs(idFilters),
		internal.WithNames(nameFilters),
		internal.WithTags(tagFilters),
	)

	req := &api.IDs{
		Ids: lo.Map(
			procIDs,
			func(procID uint64, _ int) *api.ProcessID {
				return &api.ProcessID{Id: procID}
			},
		),
	}
	if _, err := client.Stop(ctx, req); err != nil {
		return fmt.Errorf("client.Stop failed: %w", err)
	}

	if err := dbHandle.Delete(procIDs); err != nil {
		return err // TODO: add errs descriptions, loggings
	}

	for _, procID := range procIDs {
		if err := removeLogFiles(procID); err != nil {
			return err
		}
	}

	return nil
}

func removeLogFiles(procID uint64) error {
	stdoutFilename := filepath.Join(_daemonLogsDir, fmt.Sprintf("%d.stdout", procID))
	if err := removeFile(stdoutFilename); err != nil {
		return err
	}

	stderrFilename := filepath.Join(_daemonLogsDir, fmt.Sprintf("%d.stderr", procID))
	if err := removeFile(stderrFilename); err != nil {
		return err
	}

	return nil
}

func removeFile(name string) error {
	_, err := os.Stat(name)
	if err == os.ErrNotExist {
		return nil
	} else if err != nil {
		return err
	}

	return os.Remove(name)
}
