package cli

import (
	"encoding/json"
	"time"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type flagAgentConfig core.Proc

func (f *flagAgentConfig) UnmarshalFlag(value string) error {
	if err := json.Unmarshal([]byte(value), f); err != nil {
		return xerr.NewWM(err, "unmarshal agent config", xerr.Fields{"config": value})
	}

	return nil
}

type _cmdAgent struct {
	Args struct {
		Config flagAgentConfig `required:"true"`
	} `positional-args:"yes"`
}

func (x *_cmdAgent) Execute(_ []string) error {
	// TODO: remove
	// a little sleep to wait while calling process closes db file
	time.Sleep(1 * time.Second)

	app, errNewApp := app.New()
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	if err := app.StartRaw(core.Proc(x.Args.Config)); err != nil {
		return xerr.NewWM(err, "run", xerr.Fields{"arg": x.Args.Config})
	}

	return nil
}
