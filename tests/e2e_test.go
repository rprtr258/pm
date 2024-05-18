package main

import (
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/rprtr258/fun"
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
	app := useApp(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	// TODO: build server binary beforehand

	// start test processes
	id, _, errStart := cli.ImplRun(app, core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    fun.Valid("http-hello-server"),
		Command: "./hello-http/main",
		Args:    []string{":" + strconv.Itoa(serverPort)},
	})
	must.NoError(t, errStart)

	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			// check server started
			return !isTCPPortAvailable(serverPort)
		}),
		wait.Timeout(time.Second),
	))

	// check response is correct
	code, body := httpResponse(t, "http://localhost:"+strconv.Itoa(serverPort)+"/")
	must.EqOp(t, http.StatusOK, code)
	must.EqOp(t, "hello world", body)

	// stop test processes
	must.NoError(t, app.Stop(id))

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
