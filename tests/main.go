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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	mycli "github.com/rprtr258/pm/internal/infra/cli"
	"github.com/rprtr258/pm/internal/infra/errors"
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
		return errors.Wrapf(err, "create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "get response")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Newf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "read response body")
	}

	body = bytes.TrimSpace(body)
	if string(body) != expectedResponse {
		return errors.Newf("unexpected response: %s", string(body))
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
				return errors.Newf("port %d not available", _helloHTTPServerPort)
			}

			return nil
		},
		testFunc: func(ctx context.Context, client app.App) error {
			time.Sleep(time.Second)

			return httpResponse(ctx, "http://localhost:8080/", "hello world")
		},
		afterFunc: func(ctx context.Context, client app.App) error {
			if !tcpPortAvailable(_helloHTTPServerPort) {
				return errors.Newf("server not stopped, port=%d", _helloHTTPServerPort)
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
				return errors.Newf("port %d not available", _ncServerPort)
			}

			return nil
		},
		testFunc: func(ctx context.Context, client app.App) error {
			homeDir, errHome := os.UserHomeDir()
			if errHome != nil {
				return errors.Wrapf(errHome, "get home dir")
			}

			time.Sleep(time.Second)

			d, err := os.ReadFile(filepath.Join(homeDir, ".pm", "logs", "1.stdout"))
			if err != nil {
				return errors.Wrapf(err, "read server stdout")
			}

			if strings.TrimSpace(string(d)) != "123" {
				return errors.Newf("unexpected request: %s", string(d))
			}

			return nil
		},
		afterFunc: func(ctx context.Context, client app.App) error {
			if !tcpPortAvailable(_ncServerPort) {
				return errors.Newf("server not stopped, port=%d", _ncServerPort)
			}

			return nil
		},
	},
}

//nolint:nonamedreturns // required to check test result
func runTest(ctx context.Context, name string, test testcase) (ererer error) { //nolint:funlen,gocognit,lll // no idea how to refactor right now
	appp, errClient := app.New()
	if errClient != nil {
		return errors.Wrapf(errClient, "create client")
	}

	// DELETE ALL PROCS
	var err error
	appp.List()(func(proc core.Proc) bool {
		if errStop := appp.Stop(proc.ID); errStop != nil {
			err = errors.Wrapf(errStop, "stop all old processes")
			return false
		}

		if errDelete := app.Delete(appp, proc.ID); errDelete != nil {
			err = errors.Wrapf(errDelete, "delete all old processes")
			return false
		}

		return true
	})
	if err != nil {
		return err
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
			if errTest := test.beforeFunc(ctx, appp); errTest != nil {
				return errors.Wrapf(errTest, "run test before hook")
			}
		}

		// START TEST PROCESSES
		ids := []core.PMID{}
		for _, c := range test.runConfigs {
			id, errStart := appp.Run(core.RunConfig{
				Name:    c.Name,
				Command: c.Command,
				Args:    c.Args,
				Cwd:     c.Cwd,
				Tags:    c.Tags,
				Env:     c.Env,
			})
			if errStart != nil {
				return errors.Wrapf(errStart, "start process: %s", id)
			}

			ids = append(ids, core.PMID(id))
		}

		// RUN TEST
		if test.testFunc != nil {
			if errTest := test.testFunc(ctx, appp); errTest != nil {
				return errors.Wrapf(errTest, "run test func")
			}
		}

		// STOP AND REMOVE TEST PROCESSES
		for _, id := range ids {
			if errStop := appp.Stop(id); errStop != nil {
				return errors.Wrapf(errStop, "stop process: %s", id)
			}

			// TODO: block on stop method instead, now it is async
			time.Sleep(3 * time.Second)

			if errDelete := app.Delete(appp, id); errDelete != nil {
				return errors.Wrapf(errDelete, "delete process: %s", id)
			}
		}

		// RUN TEST AFTER HOOK
		if test.afterFunc != nil {
			if errTest := test.afterFunc(ctx, appp); errTest != nil {
				return errors.Wrapf(errTest, "run test after hook")
			}
		}

		return nil
	}(); errTest != nil {
		return errTest
	}

	return nil
}

var cmdTest = &cobra.Command{
	Use:   "test TESTNAME|all",
	Short: "run e2e tests",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		test := args[0]

		if test == "all" {
			for name, test := range tests {
				if errTest := runTest(ctx, name, test); errTest != nil {
					return errors.Wrapf(errTest, "run test: %s", name)
				}
			}
			return nil
		}

		testF, ok := tests[test]
		if !ok {
			return errors.Newf("unknown test: %q", test)
		}

		return runTest(ctx, test, testF)
	},
}

var _app = func() *cobra.Command {
	cmd := mycli.App
	cmd.AddCommand(cmdTest)
	return cmd
}()

func main() {
	if err := _app.Execute(); err != nil {
		log.Fatal().Err(err).Send()
	}
}
