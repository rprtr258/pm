package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/rprtr258/fun"
	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/shoenig/test/portal"
	"github.com/shoenig/test/wait"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/cli"
)

// isTCPPortAvailable checks if a given TCP port is available for use on the local network interface.
func isTCPPortAvailable(port int) bool {
	address := net.JoinHostPort("localhost", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		return true
	}
	// connected somewhere, therefore not available
	conn.Close()
	return false
}

func httpResponse(t *testing.T, endpoint string) (int, string) {
	resp, err := http.Get(endpoint)
	must.NoError(t, err, must.Sprint("get response"))
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	must.NoError(t, err, must.Sprint("read response body"))

	return resp.StatusCode, string(body)
}

func clearProcs(t *testing.T, appp app.App) {
	appp.List()(func(proc core.Proc) bool {
		must.NoError(t, appp.Stop(proc.ID))
		must.NoError(t, cli.ImplDelete(appp, proc.ID))
		return true
	})
}

type pM struct {
	t *testing.T
}

// Run returns new proc name
func (pm pM) Run(config core.RunConfig) string {
	// TODO: run from temporary config
	args := []string{}
	if config.Name.Valid {
		args = append(args, "--name", config.Name.Value)
	}
	args = append(args, append([]string{config.Command, "--"}, config.Args...)...)

	var berr bytes.Buffer

	cmd := exec.Command("./pm", append([]string{"run"}, args...)...)
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
	cmd := exec.Command("./pm", append([]string{"stop"}, selectors...)...)
	must.NoError(pm.t, cmd.Run())
}

func (pm pM) List() []core.Proc {
	logsBytes, err := exec.Command("./pm", "l", "-f", "json").Output()
	must.NoError(pm.t, err)

	var list []core.Proc
	must.NoError(pm.t, json.Unmarshal(logsBytes, &list))

	return list
}

func usePM(t *testing.T) pM {
	app, err := app.New()
	must.NoError(t, err)

	clearProcs(t, app)
	t.Cleanup(func() {
		clearProcs(t, app)
	})

	return pM{t}
}

func Test_HelloHttpServer(t *testing.T) {
	pm := usePM(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	// TODO: build server binary beforehand

	// start test processes
	name := pm.Run(core.RunConfig{
		Name:    fun.Valid("hello-http"),
		Command: "./tests/hello-http/main",
		Args:    []string{":" + strconv.Itoa(serverPort)},
	})
	must.EqOp(t, "hello-http", name)

	list := pm.List()
	test.SliceLen(t, 1, list)
	test.EqOp(t, "hello-http", list[0].Name)
	test.Eq(t, []string{"all"}, list[0].Tags)
	test.Eq(t, "/home/rprtr258/pr/pm/tests/hello-http/main", list[0].Command)
	test.Eq(t, "/home/rprtr258/pr/pm", list[0].Cwd)

	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			// check server started
			return !isTCPPortAvailable(serverPort)
		}),
		wait.Timeout(time.Second*3),
	))

	// check response is correct
	code, body := httpResponse(t, "http://localhost:"+strconv.Itoa(serverPort)+"/")
	must.EqOp(t, http.StatusOK, code)
	must.EqOp(t, "hello world", body)

	// stop test processes
	cmd2 := exec.Command("./pm", "stop", "--name", "hello-http")
	must.NoError(t, cmd2.Run())

	// check server stopped
	must.True(t, isTCPPortAvailable(serverPort))
}

func Test_ClientServerNetcat(t *testing.T) {
	t.Skip() // TODO: remove
	pm := usePM(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	//start server
	nameServer := pm.Run(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    fun.Valid("nc-server"),
		Command: "/usr/bin/nc",
		Args:    []string{"-l", "-p", strconv.Itoa(serverPort)},
	})

	// start client
	nameClient := pm.Run(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    fun.Valid("nc-client"),
		Command: "/bin/sh",
		Args:    []string{"-c", `echo "123" | nc localhost ` + strconv.Itoa(serverPort)},
	})

	homeDir, errHome := os.UserHomeDir()
	must.NoError(t, errHome, must.Sprint("get home dir"))

	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			// check server started
			return !isTCPPortAvailable(serverPort)
		}),
		wait.Timeout(time.Second),
	))

	list := pm.List()

	clientProc, _, ok := fun.Index(func(proc core.Proc) bool {
		return proc.Name == nameClient
	}, list...)
	must.True(t, ok)
	idClient := clientProc.ID

	d, err := os.ReadFile(filepath.Join(homeDir, ".pm", "logs", string(idClient)+".stdout"))
	must.NoError(t, err, must.Sprint("read server stdout"))
	must.EqOp(t, "123", string(d))

	// stop test processes
	pm.Stop(nameClient, nameServer)

	// check server stopped
	must.True(t, isTCPPortAvailable(serverPort))
}
