package internal

import (
	"fmt"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

func init() {
	AllCmds = append(AllCmds, ListCmd)
}

var ListCmd = &cli.Command{
	Name:    "list",
	Aliases: []string{"l"},
	Action: func(*cli.Context) error {
		fs, err := os.ReadDir(HomeDir)
		if err != nil {
			return err
		}

		for _, f := range fs {
			if !f.IsDir() {
				fmt.Fprintf(os.Stderr, "found strange file %q which should not exist\n", path.Join(HomeDir, f.Name()))
				continue
			}

			fmt.Printf("%#v", f.Name())
		}
		return nil
	},
}
