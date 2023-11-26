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
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	pmcli "github.com/rprtr258/pm/internal/infra/cli"
)

// tcpPortAvailable checks if a given TCP port is bound on the local network interface.
func tcpPortAvailable(port int) bool { //nolint:unparam // someday will receive something except 8080
	address := net.JoinHostPort("localhost", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}

func httpResponse(
	ctx context.Context,
	endpoint, expectedResponse string,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
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

type testHook func(ctx context.Context, client app.App) error

type testcase struct {
	beforeFunc testHook
	testFunc   testHook
	afterFunc  testHook
	runConfigs []core.RunConfig
}

const (
	_helloHTTPServerPort = 8080
	_ncServerPort        = 8080
)

var tests = map[string]testcase{
	"hello-http-server": {
		runConfigs: []core.RunConfig{{ //nolint:exhaustruct // not needed
			Name:    fun.Valid("http-hello-server"),
			Command: "./tests/hello-http/main",
		}},
		beforeFunc: func(ctx context.Context, client app.App) error {
			if !tcpPortAvailable(_helloHTTPServerPort) {
				return xerr.NewM("port not available", xerr.Fields{"port": _helloHTTPServerPort})
			}

			return nil
		},
		testFunc: func(ctx context.Context, client app.App) error {
			time.Sleep(time.Second)

			return httpResponse(ctx, "http://localhost:8080/", "hello world")
		},
		afterFunc: func(ctx context.Context, client app.App) error {
			if !tcpPortAvailable(_helloHTTPServerPort) {
				return xerr.NewM("server not stopped", xerr.Fields{"port": _helloHTTPServerPort})
			}

			return nil
		},
	},
	"client-server-netcat": {
		runConfigs: []core.RunConfig{
			{ //nolint:exhaustruct // not needed
				Name:    fun.Valid("nc-server"),
				Command: "/usr/bin/nc",
				Args:    []string{"-l", "-p", strconv.Itoa(_ncServerPort)},
			},
			{ //nolint:exhaustruct // not needed
				Name:    fun.Valid("nc-client"),
				Command: "/bin/sh",
				Args:    []string{"-c", `echo "123" | nc localhost ` + strconv.Itoa(_ncServerPort)},
			},
		},
		beforeFunc: func(ctx context.Context, client app.App) error {
			if !tcpPortAvailable(_ncServerPort) {
				return xerr.NewM("port not available", xerr.Fields{"port": _ncServerPort})
			}

			return nil
		},
		testFunc: func(ctx context.Context, client app.App) error {
			homeDir, errHome := os.UserHomeDir()
			if errHome != nil {
				return xerr.NewWM(errHome, "get home dir")
			}

			time.Sleep(time.Second)

			d, err := os.ReadFile(filepath.Join(homeDir, ".pm", "logs", "1.stdout"))
			if err != nil {
				return xerr.NewWM(err, "read server stdout")
			}

			if strings.TrimSpace(string(d)) != "123" {
				return xerr.NewM("unexpected request", xerr.Fields{"request": string(d)})
			}

			return nil
		},
		afterFunc: func(ctx context.Context, client app.App) error {
			if !tcpPortAvailable(_ncServerPort) {
				return xerr.NewM("server not stopped", xerr.Fields{"port": _ncServerPort})
			}

			return nil
		},
	},
}

//nolint:nonamedreturns // required to check test result
func runTest(ctx context.Context, name string, test testcase) (ererer error) { //nolint:funlen,gocognit,lll // no idea how to refactor right now
	client, errClient := app.New()
	if errClient != nil {
		return xerr.NewWM(errClient, "create client")
	}

	// DELETE ALL PROCS
	list := client.List()

	for id := range list {
		if errStop := client.Stop(id); errStop != nil {
			return xerr.NewWM(errStop, "stop all old processes")
		}

		if errDelete := client.Delete(id); errDelete != nil {
			return xerr.NewWM(errDelete, "delete all old processes")
		}
	}

	// RUN TEST
	if errTest := func() error {
		l := log.With().Str("test", name).Logger()

		l.Info().Msg("running test")
		defer func() {
			if ererer == nil {
				l.Info().Msg("test succeeded")
			} else {
				l.Error().Err(ererer).Msg("test failed")
			}
		}()

		// RUN TEST BEFORE HOOK
		if test.beforeFunc != nil {
			if errTest := test.beforeFunc(ctx, client); errTest != nil {
				return xerr.NewWM(errTest, "run test before hook")
			}
		}

		// START TEST PROCESSES
		ids := []core.PMID{}
		for _, c := range test.runConfigs {
			id, errStart := client.Run(core.RunConfig{
				Name:    c.Name,
				Command: c.Command,
				Args:    c.Args,
				Cwd:     c.Cwd,
				Tags:    c.Tags,
				Env:     c.Env,
			})
			if errStart != nil {
				return xerr.NewWM(errStart, "start process", xerr.Fields{"pmid": id})
			}

			ids = append(ids, core.PMID(id))
		}

		// RUN TEST
		if test.testFunc != nil {
			if errTest := test.testFunc(ctx, client); errTest != nil {
				return xerr.NewWM(errTest, "run test func")
			}
		}

		// STOP AND REMOVE TEST PROCESSES
		for _, id := range ids {
			if errStop := client.Stop(id); errStop != nil {
				return xerr.NewWM(errStop, "stop process", xerr.Fields{"pmid": id})
			}

			// TODO: block on stop method instead, now it is async
			time.Sleep(3 * time.Second)

			if errDelete := client.Delete(id); errDelete != nil {
				return xerr.NewWM(errDelete, "delete process", xerr.Fields{"pmid": id})
			}
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

	return nil
}

var (
	_testsCmds = fun.MapToSlice(
		tests,
		func(name string, test testcase) *cli.Command {
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
	// TODO: ???, why not just separate cli?
	pmcli.App.Commands = append(pmcli.App.Commands, &cli.Command{
		Name:        "test",
		Usage:       "run e2e tests",
		Subcommands: append(_testsCmds, _testAllCmd),
	})

	if err := pmcli.App.Run(os.Args); err != nil {
		log.Fatal().Err(err).Send()
	}
}
