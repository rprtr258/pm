package e2e

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

// _e2eTestDir is the directory containing the e2e tests source code
var _e2eTestDir = func() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
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

func useHTTPGet(t *testing.T, endpoint string) (int, string) {
	t.Helper()

	resp, err := http.Get(endpoint)
	must.NoError(t, err, must.Sprint("get response"))
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	test.NoError(t, err, test.Sprint("read response body"))

	return resp.StatusCode, string(body)
}

func mustExec(cmd string, args ...string) {
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
	mustExec("go", "build",
		"-cover",
		"-o", filepath.Join(_e2eTestDir, "pm"),
		filepath.Dir(_e2eTestDir))

	os.Exit(m.Run())
}

func useCwd(t testing.TB) string {
	t.Helper()
	cwd, err := os.Getwd()
	test.NoError(t, err)
	return cwd
}

func Test_HelloHttpServer(t *testing.T) { //nolint:paralleltest // not parallel
	// build server binary beforehand
	mustExec("go", "build",
		"-o", filepath.Join(_e2eTestDir, "tests", "hello-http"),
		filepath.Join(_e2eTestDir, "tests", "hello-http", "main.go"))

	pm, _ := usePM(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	// start test processes
	name := pm.UseProc(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    "hello-http",
		Command: "./tests/hello-http/main",
		Args:    []string{":" + strconv.Itoa(serverPort)},
	})
	must.EqOp(t, "hello-http", name)

	cwd := useCwd(t)

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
	code, body := useHTTPGet(t, "http://localhost:"+strconv.Itoa(serverPort)+"/")
	must.EqOp(t, http.StatusOK, code)
	must.EqOp(t, "hello world", body)

	// stop test processes
	cmd2 := exec.Command("./pm", "stop", "--name", "hello-http")
	must.NoError(t, cmd2.Run())

	// check server stopped
	test.True(t, isTCPPortAvailableForListen(serverPort))

	must.NoError(t, pm.Delete("hello-http")) // TODO: make ficture
}

func Test_ClientServerNetcat(t *testing.T) { //nolint:paralleltest // not parallel
	pm, dataDir := usePM(t)

	serverPort := portal.New(t, portal.WithAddress("localhost")).One()

	// start server
	must.EqOp(t, "nc-server", pm.UseProc(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    "nc-server",
		Command: "/usr/bin/nc",
		Args:    []string{"-l", "-p", strconv.Itoa(serverPort)},
	}))

	time.Sleep(3 * time.Second)
	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			return !isTCPPortAvailableForListen(serverPort)
		}),
		wait.Timeout(10*time.Second),
		wait.Gap(3*time.Second),
	), must.Sprint("check server started"))

	// start client
	must.EqOp(t, "nc-client", pm.UseProc(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:    "nc-client",
		Command: "/bin/sh",
		Args:    []string{"-c", `echo "123" | nc localhost ` + strconv.Itoa(serverPort)},
	}))

	list := pm.List()

	serverProc, _, ok := fun.Index(func(proc core.ProcStat) bool {
		return proc.Name == "nc-server"
	}, list...)
	must.True(t, ok)
	serverID := serverProc.ID

	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			d, err := os.ReadFile(filepath.Join(dataDir, "pm", "logs", string(serverID)+".stdout"))
			test.NoError(t, err, test.Sprint("read server stdout"))
			return string(d) == "123\r\n"
		}),
		wait.Timeout(time.Second*10),
	), must.Sprint("check server received payload"))

	// stop test processes
	must.NoError(t, pm.Stop("nc-client", "nc-server"))

	// check server stopped
	test.True(t, isTCPPortAvailableForListen(serverPort))
	must.NoError(t, pm.Delete("nc-client", "nc-server"))
}

func Test_MaxRestarts(t *testing.T) {
	// build binary beforehand
	mustExec("go", "build",
		"-o", filepath.Join(_e2eTestDir, "tests", "crashloop"),
		filepath.Join(_e2eTestDir, "tests", "crashloop", "main.go"))

	pm, dataDir := usePM(t)

	const restarts = 3
	must.EqOp(t, "mx", pm.UseProc(core.RunConfig{ //nolint:exhaustruct // not needed
		Name:        "mx",
		Command:     "./tests/crashloop/main",
		Args:        []string{},
		MaxRestarts: restarts,
	}))

	must.Wait(t, wait.InitialSuccess(
		wait.BoolFunc(func() bool {
			list := pm.List()
			return len(list) == 1 && list[0].Name == "mx" && list[0].Status == core.StatusStopped
		}),
		wait.Timeout((restarts+2)*time.Second),
		wait.Gap(500*time.Millisecond),
	), must.Sprint("check proc stopped"))

	list := pm.List()

	proc, _, ok := fun.Index(func(proc core.ProcStat) bool {
		return proc.Name == "mx"
	}, list...)
	must.True(t, ok)

	logs, err := os.ReadFile(filepath.Join(dataDir, "pm", "logs", string(proc.ID)+".stdout"))
	test.NoError(t, err, test.Sprint("read stdout"))
	must.Eq(t, strings.Repeat("trying to wake up\r\nnah, going back to sleep\r\n", restarts+1), string(logs))
}
