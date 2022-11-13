package internal

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/db"
	"github.com/urfave/cli/v2"
)

func init() {
	AllCmds = append(AllCmds, DeleteCmd)
}

var DeleteCmd = &cli.Command{
	Name:      "delete",
	Aliases:   []string{"del", "rm"},
	Usage:     "stop and remove process",
	ArgsUsage: "<name|id|namespace|tag|json>...",
	Action: func(ctx *cli.Context) error {
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferErr(deferFunc)

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("what to delete expected")
		}

		procIDs := []uint64{} // TODO: implement

		for _, arg := range args {
			procID, err := strconv.ParseUint(arg, 10, 64)
			if err != nil {
				return fmt.Errorf("%q is not uint: %w", arg, err)
			}

			procIDs = append(procIDs, procID)
		}

		if _, err := client.Stop(ctx.Context, &api.IDs{Ids: procIDs}); err != nil {
			return fmt.Errorf("client.Stop failed: %w", err)
		}

		if err := db.New(_daemonDBFile).Delete(procIDs); err != nil {
			return err // TODO: add errs descriptions, loggings
		}

		// TODO: delete log files too
		// 	_, err := os.Stat(string(*f))
		// 	if err != nil {
		// 		return true
		// 	}
		// 	err = os.Remove(string(*f))
		// 	return err == nil
		// }

		return nil
	},
}
