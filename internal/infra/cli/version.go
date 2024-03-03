package cli

import (
	"context"
	"fmt"

	"github.com/rprtr258/pm/internal/core"
)

type _cmdVersion struct{}

func (_cmdVersion) Execute(ctx context.Context) error {
	fmt.Println(core.Version)
	return nil
}
