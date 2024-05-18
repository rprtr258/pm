package main

import (
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

func useApp(t *testing.T) app.App {
	appp, err := app.New()
	must.NoError(t, err)

	clearProcs(t, appp)
	t.Cleanup(func() {
		clearProcs(t, appp)
	})

	return appp
}

func Test_HelloHttpServer(t *testing.T) {
	useApp(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	// TODO: build server binary beforehand

	// start test processes
	cmd := exec.Command("./pm", "run", "--name", "hello-http", "./tests/hello-http/main", ":"+strconv.Itoa(serverPort))
	nameBytes, err := cmd.Output()
	must.NoError(t, err)
	must.EqOp(t, "hello-http\n", string(nameBytes))

	cmd3 := exec.Command("./pm", "l", "-f", "json")
	logsBytes, err := cmd3.Output()
	test.NoError(t, err)
	var list []core.Proc
	test.NoError(t, json.Unmarshal(logsBytes, &list))
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
	app := useApp(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	//start server
	idServer, _, errStart := cli.ImplRun(app, core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    fun.Valid("nc-server"),
		Command: "/usr/bin/nc",
		Args:    []string{"-l", "-p", strconv.Itoa(serverPort)},
	})
	must.NoError(t, errStart)

	// start client
	idClient, _, errStart := cli.ImplRun(app, core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    fun.Valid("nc-client"),
		Command: "/bin/sh",
		Args:    []string{"-c", `echo "123" | nc localhost ` + strconv.Itoa(serverPort)},
	})
	must.NoError(t, errStart)

	homeDir, errHome := os.UserHomeDir()
	must.NoError(t, errHome, must.Sprint("get home dir"))

	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			// check server started
			return !isTCPPortAvailable(serverPort)
		}),
		wait.Timeout(time.Second),
	))

	d, err := os.ReadFile(filepath.Join(homeDir, ".pm", "logs", string(idClient)+".stdout"))
	must.NoError(t, err, must.Sprint("read server stdout"))

	must.EqOp(t, "123", string(d))

	// stop test processes
	must.NoError(t, app.Stop(idServer))

	// check server stopped
	must.True(t, isTCPPortAvailable(serverPort))
}
