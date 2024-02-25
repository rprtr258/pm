package cli

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rprtr258/pm/internal/infra/errors"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type flagAgentConfig core.Proc

func (f *flagAgentConfig) UnmarshalFlag(value string) error {
	if err := json.Unmarshal([]byte(value), f); err != nil {
		return errors.Wrap(err, "unmarshal agent config: %s", value)
	}

	return nil
}

type _cmdAgent struct {
	Args struct {
		Config flagAgentConfig `required:"true"`
	} `positional-args:"yes"`
}

func (x _cmdAgent) Execute(ctx context.Context) error {
	// TODO: remove
	// a little sleep to wait while calling process closes db file
	time.Sleep(1 * time.Second)

	app, errNewApp := app.New()
	if errNewApp != nil {
		return errors.Wrap(errNewApp, "new app")
	}

	if err := app.StartRaw(core.Proc(x.Args.Config)); err != nil {
		return errors.Wrap(err, "run: %v", x.Args.Config)
	}

	return nil
}
