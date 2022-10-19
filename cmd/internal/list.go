package internal

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	AllCmds = append(AllCmds, ListCmd)
}

var ListCmd = &cli.Command{
	Name:    "list",
	Aliases: []string{"l"},
	Action: func(ctx *cli.Context) error {
		client, deferFunc, err := NewGrpcClient()
		if err != nil {
			return err
		}
		defer deferFunc()

		resp, err := client.List(ctx.Context, &emptypb.Empty{})
		if err != nil {
			return err
		}

		for _, item := range resp.GetItems() {
			fmt.Printf("%#v\n", item)
		}

		return nil
	},
}
