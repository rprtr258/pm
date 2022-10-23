package internal

import (
	"errors"

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
		// client, deferFunc, err := NewGrpcClient()
		// if err != nil {
		// 	return err
		// }
		// defer deferFunc()

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("what to delete expected")
		}

		// TODO: stop
		// for _, id := range resp.GetId() {
		// 	fmt.Println(id)
		// }
		return nil
	},
}
