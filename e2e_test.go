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
	"github.com/rs/zerolog/log"
	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/shoenig/test/portal"
	"github.com/shoenig/test/skip"
	"github.com/shoenig/test/wait"

	"github.com/rprtr258/pm/internal/core"
)

var homeDir = func() string {
	res, err := os.UserHomeDir()
	if err != nil {
		panic(err.Error())
	}
	return res
}()

// isTCPPortAvailable checks if a given TCP port is available for use on the local network interface
func isTCPPortAvailableForListen(port int) bool {
	conn, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: port,
	})
	if err == nil {
		return true
	}
	// socket created, hence port is still free
	conn.Close()
	return false
}

// isTCPPortAvailable checks if a given TCP port is available for use on the local network interface
func isTCPPortAvailableForDial(port int) bool {
	conn, err := net.Dial("tcp", net.JoinHostPort("localhost", strconv.Itoa(port)))
	if err != nil {
		return true
	}
	// socket created, hence port is still free
	conn.Close()
	return false
}

func httpResponse(t *testing.T, endpoint string) (int, string) {
	t.Helper()

	resp, err := http.Get(endpoint)
	must.NoError(t, err, must.Sprint("get response"))
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	test.NoError(t, err, test.Sprint("read response body"))

	return resp.StatusCode, string(body)
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

func (pM) delete(selectors ...string) error {
	return exec.Command("./pm", append([]string{"rm"}, selectors...)...).Run()
}

func (pm pM) Delete(selectors ...string) {
	must.NoError(pm.t, pm.delete(selectors...))
}

func (pm pM) List() []core.Proc {
	logsBytes, err := exec.Command("./pm", "l", "-f", "json").Output()
	must.NoError(pm.t, err)

	var list []core.Proc
	must.NoError(pm.t, json.Unmarshal(logsBytes, &list))

	return list
}

func TestMain(m *testing.M) {
	pm := pM{}
	// TODO: backup old pm state
	if err := pm.delete("all"); err != nil {
		log.Fatal().Err(err).Send()
	}
	code := m.Run()
	// TODO: restore old pm state
	if err := pm.delete("all"); err != nil {
		log.Fatal().Err(err).Send()
	}
	os.Exit(code)
}

func usePM(t *testing.T) pM {
	pm := pM{t}

	t.Cleanup(func() {
		pm.Delete("all")
	})

	return pm
}

func Test_HelloHttpServer(t *testing.T) {
	skip.CommandUnavailable(t, "./tests/hello-http/main")

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
	test.StrHasSuffix(t, "pm/tests/hello-http/main", list[0].Command)
	test.StrHasSuffix(t, "pm", list[0].Cwd)

	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			// check server started
			return !isTCPPortAvailableForDial(serverPort)
		}),
		wait.Timeout(time.Second*5),
	))

	// check response is correct
	code, body := httpResponse(t, "http://localhost:"+strconv.Itoa(serverPort)+"/")
	must.EqOp(t, http.StatusOK, code)
	must.EqOp(t, "hello world", body)

	// stop test processes
	cmd2 := exec.Command("./pm", "stop", "--name", "hello-http")
	must.NoError(t, cmd2.Run())

	// check server stopped
	must.True(t, isTCPPortAvailableForDial(serverPort))
}

func Test_ClientServerNetcat(t *testing.T) {
	pm := usePM(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	// start server
	must.EqOp(t, "nc-server", pm.Run(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    fun.Valid("nc-server"),
		Command: "/usr/bin/nc",
		Args:    []string{"-l", "-p", strconv.Itoa(serverPort)},
	}))

	time.Sleep(3 * time.Second)
	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			return !isTCPPortAvailableForListen(serverPort)
		}),
		wait.Timeout(time.Second*10),
		wait.Gap(3*time.Second),
	), must.Sprint("check server started"))

	// start client
	must.EqOp(t, "nc-client", pm.Run(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    fun.Valid("nc-client"),
		Command: "/bin/sh",
		Args:    []string{"-c", `echo "123" | nc localhost ` + strconv.Itoa(serverPort)},
	}))

	list := pm.List()

	serverProc, _, ok := fun.Index(func(proc core.Proc) bool {
		return proc.Name == "nc-server"
	}, list...)
	must.True(t, ok)
	serverID := serverProc.ID

	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			d, err := os.ReadFile(filepath.Join(homeDir, ".pm", "logs", string(serverID)+".stdout"))
			test.NoError(t, err, test.Sprint("read server stdout"))
			return string(d) == "123\n"
		}),
		wait.Timeout(time.Second*10),
	), must.Sprint("check server received payload"))

	// stop test processes
	pm.Stop("nc-client", "nc-server")

	// check server stopped
	must.True(t, isTCPPortAvailableForListen(serverPort))
}
