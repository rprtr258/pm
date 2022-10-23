package internal

import (
	"errors"
	"fmt"

	"github.com/rprtr258/pm/api"
	"github.com/urfave/cli/v2"
)

func init() {
	AllCmds = append(AllCmds, DeleteCmd)
}

var DeleteCmd = &cli.Command{
	Name:      "delete",
	Usage:     "stop and remove process",
	ArgsUsage: "<id|name|namespace|all|json>...",
	Action: func(ctx *cli.Context) error {
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferFunc()

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("what to delete expected")
		}

		resp, err := client.Delete(ctx.Context, &api.DeleteReq{
			Filters: []*api.DeleteFilter{{
				// Filter: &api.DeleteFilter_Name{Name: name},
				// Filter: &api.DeleteFilter_Tags{Tags: &api.Tags{Tags: []string{}}},
			}},
		})
		if err != nil {
			return err
		}

		for _, id := range resp.GetId() {
			fmt.Println(id)
		}
		return nil
	},
}
