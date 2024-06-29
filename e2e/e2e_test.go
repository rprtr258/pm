package e2e

import (
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
		Zone: "",
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

func testMain(m *testing.M) int {
	pm := pM{t: nil}

	{
		// backup pm dir if exists
		oldDirExists := true
		pmDirOld := filepath.Join(homeDir, ".pm")
		pmDirBackup := filepath.Join(homeDir, ".pm.bak")
		if err := os.Rename(pmDirOld, pmDirBackup); err != nil {
			if os.IsNotExist(err) {
				oldDirExists = false
			} else {
				log.Error().Err(err).Msg("backup pm dir")
			}
		}
		// restore pm dir backup
		if oldDirExists {
			defer func() {
				if err := os.RemoveAll(pmDirOld); err != nil {
					log.Fatal().Err(err).Msg("remove pm dir")
				}
				if err := os.Rename(pmDirBackup, pmDirOld); err != nil {
					log.Fatal().Err(err).Msg("restore pm dir backup")
				}
			}()
		}
	}

	if err := pm.delete("all"); err != nil {
		log.Error().Err(err).Msg("clear old processes")
	}
	defer func() {
		if err := pm.delete("all"); err != nil {
			log.Error().Err(err).Msg("clear tested processes")
		}
	}()

	return m.Run()
}

func exec_(cmd string, args ...string) {
	if err := exec.Command(cmd, args...).Run(); err != nil {
		log.Fatal().
			Err(err).
			Str("cmd", cmd).
			Strs("args", args).
			Msg("failed to exec")
	}
}

func TestMain(m *testing.M) {
	// build pm binary
	exec_("go", "build", "-o", "e2e/pm", ".")

	os.Exit(testMain(m))
}

func Test_HelloHttpServer(t *testing.T) { //nolint:paralleltest // not parallel
	// build server binary beforehand
	exec_("go", "build", "-o", "tests/hello-http", "tests/hello-http/main.go")

	pm := usePM(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	// start test processes
	name := pm.Run(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    fun.Valid("hello-http"),
		Command: "./tests/hello-http/main",
		Args:    []string{":" + strconv.Itoa(serverPort)},
	})
	must.EqOp(t, "hello-http", name)

	cwd, err := os.Getwd()
	test.NoError(t, err)

	list := pm.List()
	test.SliceLen(t, 1, list)
	test.EqOp(t, "hello-http", list[0].Name)
	test.Eq(t, []string{"all"}, list[0].Tags)
	test.StrHasSuffix(t, "e2e/tests/hello-http/main", list[0].Command)
	test.EqOp(t, cwd, list[0].Cwd)

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
	test.True(t, isTCPPortAvailableForListen(serverPort))
}

func Test_ClientServerNetcat(t *testing.T) { //nolint:paralleltest // not parallel
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
	test.True(t, isTCPPortAvailableForListen(serverPort))
}
