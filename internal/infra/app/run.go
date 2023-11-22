package app

import (
	"os/exec"
	"path/filepath"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
)

// Run - create and start processes, returns ids of created processes.
// ids must be handled before handling error, because it tries to run all
// processes and error contains info about all failed processes, not only first.
func (app App) Run(config core.RunConfig) (core.PMID, error) {
	command, errLook := exec.LookPath(config.Command)
	if errLook != nil {
		// if command is relative and failed to look it up, add workdir first
		if filepath.IsLocal(config.Command) {
			config.Command = filepath.Join(config.Cwd, config.Command)
		}

		command, errLook = exec.LookPath(config.Command)
		if errLook != nil {
			return "", xerr.NewWM(
				errLook,
				"look for executable path",
				xerr.Fields{"executable": config.Command},
			)
		}
	}

	var merr error
	if command == config.Command { // command contains slash and might be relative
		var errAbs error
		command, errAbs = filepath.Abs(command)
		if errAbs != nil {
			xerr.AppendInto(&merr, xerr.NewWM(
				errAbs,
				"abs",
				xerr.Fields{"command": command},
			))
		}
	}

	request := core.RunConfig{
		Command:    command,
		Args:       config.Args,
		Name:       config.Name,
		Cwd:        config.Cwd,
		Tags:       config.Tags,
		Env:        config.Env,
		Watch:      config.Watch,
		StdoutFile: config.StdoutFile,
		StderrFile: config.StdoutFile,
	}
	id, errCreate := app.Create(request)
	if errCreate != nil {
		return "", xerr.NewWM(
			errCreate,
			"server.create",
			xerr.Fields{"process_options": request},
		)
	}

	if errStart := app.startAgent(id); errStart != nil {
		return core.PMID(id), xerr.NewWM(errStart, "start processes", xerr.Errors{merr})
	}

	return core.PMID(id), merr
}
