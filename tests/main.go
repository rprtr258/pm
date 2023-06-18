package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon"
	pmcli "github.com/rprtr258/pm/internal/infra/cli"
	pmclient "github.com/rprtr258/pm/pkg/client"
)

// tcpPortAvailable checks if a given TCP port is bound on the local network interface.
func tcpPortAvailable(port int) bool {
	address := net.JoinHostPort("localhost", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, time.Second) //nolint:gomnd // arbitrary time
	if err != nil {
		return true
	}
	defer conn.Close()

	return false
}

func httpResponse(
	ctx context.Context,
	endpoint, expectedResponse string,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return xerr.NewWM(err, "create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return xerr.NewWM(err, "get response")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return xerr.NewM("bad status code", xerr.Fields{"status_code": resp.StatusCode})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return xerr.NewWM(err, "read response body")
	}

	body = bytes.TrimSpace(body)
	if string(body) != expectedResponse {
		return xerr.NewM("unexpected response", xerr.Fields{"response": string(body)})
	}

	return nil
}

var client pmclient.Client

type testHook func(ctx context.Context, client pmclient.Client) error

type testcase struct {
	beforeFunc testHook
	testFunc   testHook
	afterFunc  testHook
	runConfigs []core.RunConfig
}

var tests = map[string]testcase{
	"hello-http-server": {
		runConfigs: []core.RunConfig{{
			Name:    fun.Valid("http-hello-server"),
			Command: "/home/rprtr258/.gvm/gos/go1.19.5/bin/go",
			Args:    []string{"run", "tests/hello-http/main.go"},
		}},
		beforeFunc: func(ctx context.Context, client pmclient.Client) error {
			if !tcpPortAvailable(8080) {
				return xerr.NewM("port not available", xerr.Fields{"port": 8080})
			}

			return nil
		},
		testFunc: func(ctx context.Context, client pmclient.Client) error {
			time.Sleep(time.Second)

			if errHTTP := httpResponse(ctx, "http://localhost:8080/", "hello world"); errHTTP != nil {
				return errHTTP
			}

			return nil
		},
		afterFunc: func(ctx context.Context, client pmclient.Client) error {
			if !tcpPortAvailable(8080) {
				return xerr.NewM("server not stopped", xerr.Fields{"port": 8080})
			}

			return nil
		},
	},
	"client-server-netcat": {
		runConfigs: []core.RunConfig{
			{
				Name:    fun.Valid("nc-server"),
				Command: "/usr/bin/nc",
				Args:    []string{"-l", "-p", "8080"},
			},
			{
				Name:    fun.Valid("nc-client"),
				Command: "/usr/bin/sh",
				Args:    []string{"-c", `echo "123" | nc localhost 8080`},
			},
		},
		beforeFunc: func(ctx context.Context, client pmclient.Client) error {
			if !tcpPortAvailable(8080) {
				return xerr.NewM("port not available", xerr.Fields{"port": 8080})
			}

			return nil
		},
		testFunc: func(ctx context.Context, client pmclient.Client) error {
			homeDir, errHome := os.UserHomeDir()
			if errHome != nil {
				return xerr.NewWM(errHome, "get home dir")
			}

			time.Sleep(time.Second)

			d, err := os.ReadFile(filepath.Join(homeDir, ".pm/logs/1.stdout"))
			if err != nil {
				return xerr.NewWM(err, "read server stdout")
			}

			if strings.TrimSpace(string(d)) != "123" {
				return xerr.NewM("unexpected request", xerr.Fields{"request": string(d)})
			}

			return nil
		},
		afterFunc: func(ctx context.Context, client pmclient.Client) error {
			if !tcpPortAvailable(8080) {
				return xerr.NewM("server not stopped", xerr.Fields{"port": 8080})
			}

			return nil
		},
	},
}

//nolint:nonamedreturns // required to check test result
func runTest(ctx context.Context, name string, test testcase) (ererer error) {
	var errClient error
	client, errClient = pmclient.NewGrpcClient()
	if errClient != nil {
		return xerr.NewWM(errClient, "create client")
	}

	// START DAEMON
	log.Debug("starting daemon...")
	_, errRestart := daemon.Restart(ctx)
	if errRestart != nil {
		return xerr.NewWM(errRestart, "restart daemon")
	}

	ticker := time.NewTicker(100 * time.Millisecond) //nolint:gomnd // arbitrary time
	defer ticker.Stop()
	for {
		if client.HealthCheck(ctx) == nil {
			break
		}

		select {
		case <-ctx.Done():
			return xerr.NewWM(ctx.Err(), "context done while waiting for daemon to start")
		case <-ticker.C:
		}
	}

	// DELETE ALL PROCS
	list, errList := client.List(ctx)
	if errList != nil {
		return xerr.NewWM(errList, "list processes")
	}

	oldIDs := fun.Map(fun.Keys(list), func(id core.ProcID) uint64 {
		return uint64(id)
	})

	if _, errStop := client.Stop(ctx, oldIDs); errStop != nil {
		return xerr.NewWM(errStop, "stop all old processes")
	}

	if errDelete := client.Delete(ctx, oldIDs); errDelete != nil {
		return xerr.NewWM(errDelete, "delete all old processes")
	}

	// RUN TEST
	if errTest := func() error {
		log.Infof("running test", log.F{"test": name})
		defer func() {
			if ererer == nil {
				log.Infof("test succeeded", log.F{"test": name})
			} else {
				log.Errorf("test failed", log.F{"test": name, "err": ererer.Error()})
			}
		}()

		// RUN TEST BEFORE HOOK
		if test.beforeFunc != nil {
			if errTest := test.beforeFunc(ctx, client); errTest != nil {
				return xerr.NewWM(errTest, "run test before hook")
			}
		}

		// START TEST PROCESSES
		processesOptions := fun.Map(test.runConfigs, func(c core.RunConfig) *api.ProcessOptions {
			return &api.ProcessOptions{
				Name:    c.Name.Ptr(),
				Command: c.Command,
				Args:    c.Args,
				Cwd:     c.Cwd,
				Tags:    c.Tags,
			}
		})

		ids, errCreate := client.Create(ctx, processesOptions)
		if errCreate != nil {
			return xerr.NewWM(errCreate, "create process")
		}

		if len(ids) != len(test.runConfigs) {
			return xerr.NewM("unexpected number of processes", xerr.Fields{"ids": ids})
		}

		if errStart := client.Start(ctx, ids); errStart != nil {
			return xerr.NewWM(errStart, "start process", xerr.Fields{"ids": ids})
		}

		// RUN TEST
		if test.testFunc != nil {
			if errTest := test.testFunc(ctx, client); errTest != nil {
				return xerr.NewWM(errTest, "run test func")
			}
		}

		// STOP AND REMOVE TEST PROCESSES
		if _, errStop := client.Stop(ctx, ids); errStop != nil {
			return xerr.NewWM(errStop, "stop process", xerr.Fields{"ids": ids})
		}

		if errDelete := client.Delete(ctx, ids); errDelete != nil {
			return xerr.NewWM(errDelete, "delete process", xerr.Fields{"ids": ids})
		}

		// RUN TEST AFTER HOOK
		if test.afterFunc != nil {
			if errTest := test.afterFunc(ctx, client); errTest != nil {
				return xerr.NewWM(errTest, "run test after hook")
			}
		}

		return nil
	}(); errTest != nil {
		return errTest
	}

	// KILL DAEMON
	log.Debug("killing daemon...")
	if errKill := daemon.Kill(); errKill != nil {
		return xerr.NewWM(errKill, "kill daemon")
	}

	if errHealth := client.HealthCheck(ctx); errHealth == nil {
		return xerr.NewM("daemon is healthy but must not")
	}

	return nil
}

var (
	_testsCmds = fun.ToSlice(tests, func(name string, test testcase) *cli.Command {
		return &cli.Command{
			Name: name,
			Action: func(ctx *cli.Context) error {
				return runTest(ctx.Context, name, test)
			},
		}
	})
	_testAllCmd = &cli.Command{
		Name: "all",
		Action: func(ctx *cli.Context) error {
			for name, test := range tests {
				if errTest := runTest(ctx.Context, name, test); errTest != nil {
					return xerr.NewWM(errTest, "run test", xerr.Fields{"test": name})
				}
			}

			return nil
		},
	}
)

func main() {
	pmcli.App.Commands = append(pmcli.App.Commands, &cli.Command{
		Name:        "test",
		Usage:       "run e2e tests",
		Subcommands: append(_testsCmds, _testAllCmd),
	})

	if err := pmcli.App.Run(os.Args); err != nil {
		log.Fatal(err.Error())
	}
}
