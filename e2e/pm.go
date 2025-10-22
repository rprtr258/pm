package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/shoenig/test/must"

	"github.com/rprtr258/pm/internal/core"
)

const _pmBin = "./pm"

type pM struct {
	t *testing.T
}

func usePM(t *testing.T) pM {
	t.Helper()

	pm := pM{t}
	t.Cleanup(func() {
		must.NoError(t, pm.Delete("all"))
	})
	return pm
}

func (pM) exec(cmd string, args ...string) *exec.Cmd {
	return exec.CommandContext( //nolint:gosec // fuck you, _pmBin is constant
		context.Background(),
		_pmBin,
		append([]string{cmd}, args...)...,
	)
}

// Run returns new proc name
func (pm pM) Run(config core.RunConfig) string {
	args := []string{}
	args = append(args, "--name", config.Name)
	for _, tag := range config.Tags {
		args = append(args, "--tag", tag)
	}
	for k, v := range config.Env {
		args = append(args, "--env", k+"="+v)
	}
	if config.Watch.Valid {
		args = append(args, "--watch", config.Watch.Value.String())
	}
	if config.StdoutFile.Valid {
		args = append(args, "--stdout", config.StdoutFile.Value)
	}
	if config.StderrFile.Valid {
		args = append(args, "--stderr", config.StderrFile.Value)
	}
	if config.Cwd != "" {
		args = append(args, "--cwd", config.Cwd)
	}
	args = append(args, append([]string{config.Command, "--"}, config.Args...)...)

	var berr bytes.Buffer

	cmd := pm.exec("run", args...)
	cmd.Stderr = &berr
	nameBytes, err := cmd.Output()
	must.NoError(pm.t, err)

	if berr.String() != "" {
		pm.t.Fatal(berr.String())
	}
	must.NonZero(pm.t, len(nameBytes))

	// cut newline
	return string(nameBytes[:len(nameBytes)-1])
}

func (pm pM) Stop(selectors ...string) error {
	return pm.exec("stop", selectors...).Run()
}

func (pm pM) delete(selectors ...string) error {
	return pm.exec("rm", selectors...).Run()
}

func (pm pM) Delete(selectors ...string) error {
	return pm.delete(selectors...)
}

func (pm pM) List() []core.Proc {
	cmd := pm.exec("l", "-f", "json")
	cmd.Stderr = os.Stderr

	logsBytes, err := cmd.Output()
	must.NoError(pm.t, err)

	var list []core.Proc
	must.NoError(pm.t, json.Unmarshal(logsBytes, &list), must.Values("output", string(logsBytes)))

	return list
}
