package e2e

import (
	"bytes"
	"encoding/json"
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
		pm.Delete("all")
	})
	return pm
}

// Run returns new proc name
func (pm pM) Run(config core.RunConfig) string {
	args := []string{}
	if config.Name.Valid {
		args = append(args, "--name", config.Name.Value)
	}
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

	cmd := exec.Command(_pmBin, append([]string{"run"}, args...)...)
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

func (pm pM) Stop(selectors ...string) {
	cmd := exec.Command(_pmBin, append([]string{"stop"}, selectors...)...)
	must.NoError(pm.t, cmd.Run())
}

func (pM) delete(selectors ...string) error {
	return exec.Command(_pmBin, append([]string{"rm"}, selectors...)...).Run()
}

func (pm pM) Delete(selectors ...string) {
	must.NoError(pm.t, pm.delete(selectors...))
}

func (pm pM) List() []core.Proc {
	logsBytes, err := exec.Command(_pmBin, "l", "-f", "json").Output()
	must.NoError(pm.t, err)

	var list []core.Proc
	must.NoError(pm.t, json.Unmarshal(logsBytes, &list))

	return list
}