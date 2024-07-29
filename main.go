package main

import (
	"os"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		_ = err // NOTE: ignore, since cobra will print the error
	}
}
