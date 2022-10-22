package internal

import (
	"fmt"
	"os"

	"github.com/aquasecurity/table"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/rprtr258/pm/api"
)

func init() {
	AllCmds = append(AllCmds, ListCmd)
}

func mapStatus(pbStatus any) string {
	switch status := pbStatus.(type) {
	case *pb.ListRespEntry_Running:
		return fmt.Sprintf(
			"running(pid=%d,uptime=%v)",
			status.Running.GetPid(),
			status.Running.GetUptime(),
		)
	case *pb.ListRespEntry_Stopped:
		return "stopped"
	case *pb.ListRespEntry_Errored:
		return "errored"
	case *pb.ListRespEntry_Invalid:
		return fmt.Sprintf("invalid(%T)", status)
	default:
		return fmt.Sprintf("BROKEN(%T)", status)
	}
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

		t := table.New(os.Stdout)

		t.SetHeaders("id", "name", "status", "tags", "cpu", "memory", "cmd")

		lo.ForEach(resp.GetItems(), func(item *pb.ListRespEntry, _ int) {
			t.AddRow(
				fmt.Sprint(item.GetId()),
				item.GetName(),
				mapStatus(item.GetStatus()),
				fmt.Sprint(item.GetTags().GetTags()),
				fmt.Sprint(item.GetCpu()),
				fmt.Sprint(item.GetMemory()),
				item.GetCmd(),
			)
		})

		t.Render()

		return nil
	},
}
