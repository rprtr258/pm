package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/shoenig/test"
	"github.com/shoenig/test/must"

	"github.com/rprtr258/pm/internal/core"
)

const _pmBin = "./pm"

type pM struct {
	t *testing.T
}

func useTempDir(t testing.TB, pattern string) string {
	t.Helper()

	dir, err := os.MkdirTemp(os.TempDir(), pattern)
	test.NoError(t, err, test.Sprint("create temp dir"))
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Warn().Err(err).Msg("remove pm dir")
		}
	})
	return dir
}

func usePM(t *testing.T) (pM, string) {
	t.Helper()

	dataDir := useTempDir(t, "pm-e2e-test-*")
	// TODO: expose var constant from xdg lib?
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("XDG_CONFIG_HOME", dataDir)

	pm := pM{t}

	t.Cleanup(func() {
		// ensure no procs left
		list := pm.List()
		if len(list) == 0 {
			return
		}

		t.Errorf("procs left: %v", list)
		test.NoError(t, pm.delete("all"), test.Sprint("clear old processes"))
		// time.Sleep(3 * time.Second)
	})

	return pm, dataDir
}

func (p pM) exec(cmd string, args ...string) *exec.Cmd {
	return exec.CommandContext( //nolint:gosec // fuck you, _pmBin is constant
		context.TODO(),
		_pmBin,
		append([]string{cmd}, args...)...,
	)
}

// Run returns new proc name
func (pm pM) run(config core.RunConfig) string {
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
	if config.MaxRestarts != 0 {
		args = append(args, "--max-restarts", strconv.Itoa(int(config.MaxRestarts)))
	}
	args = append(args, append([]string{config.Command, "--"}, config.Args...)...)

	var berr bytes.Buffer

	cmd := pm.exec("run", args...)
	cmd.Stderr = &berr
	nameBytes, err := cmd.Output()
	must.NoError(pm.t, err)

	if berr.String() != "" {
		pm.t.Log("STDERR:", berr.String())
	}

	name, ok := strings.CutSuffix(string(nameBytes), "\n")
	must.True(pm.t, ok)

	return name
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

func (pm pM) List() []core.ProcStat {
	cmd := pm.exec("l", "-f", "json")
	cmd.Stderr = os.Stderr

	logsBytes, err := cmd.Output()
	must.NoError(pm.t, err)

	var list []core.ProcStat
	must.NoError(pm.t, json.Unmarshal(logsBytes, &list), must.Values("output", string(logsBytes)))

	return list
}

func (pm pM) UseProc(config core.RunConfig) string {
	t := pm.t
	name := pm.run(config)
	t.Cleanup(func() {
		must.NoError(t, pm.Delete("mx"))
	})
	return name
}
