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
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "mini-list",
			Aliases: []string{"m"},
			Usage:   "display a compacted list without formatting",
		},
		&cli.BoolFlag{
			Name:  "sort",
			Usage: "sort <id|name|pid>:<inc|dec> sort process according to field value",
		},
		&cli.StringFlag{
			Name:    "format",
			Aliases: []string{"f"},
			Usage:   "Go template string to use for formatting",
		},
	},
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

		fmt.Println("id\tname\tstatus\ttags\tcpu\tmemory\tcmd")
		for _, item := range resp.GetItems() {
			fmt.Printf(
				"%d\t%s\t%T\t%v\t%d\t%d\t%s\n",
				item.GetId(),
				item.GetName(),
				item.GetStatus(),
				item.GetTags().GetTags(),
				item.GetCpu(),
				item.GetMemory(),
				item.GetCmd(),
			)
		}

		return nil
	},
}
