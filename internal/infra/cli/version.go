package cli

import (
	"fmt"

	"github.com/rprtr258/pm/internal/core"
)

type _cmdVersion struct{}

func (*_cmdVersion) Execute(_ []string) error {
	fmt.Println(core.Version)
	return nil
}
