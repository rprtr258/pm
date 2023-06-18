package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
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

// TCPPortAvailable checks if a given TCP port is bound on the local network interface.
func TCPPortAvailable(t *testing.T, port int, timeout time.Duration) bool {
	t.Helper()

	address := net.JoinHostPort("localhost", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}

func HTTPResponse(
	ctx context.Context,
	endpoint, expectedResponse string,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return xerr.NewWM(err, "failed to create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return xerr.NewWM(err, "failed to get response")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return xerr.NewM("bad status code", xerr.Fields{"status_code": resp.StatusCode})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return xerr.NewWM(err, "failed to read response body")
	}

	body = bytes.TrimSpace(body)
	if string(body) != expectedResponse {
		return xerr.NewM("unexpected response", xerr.Fields{"response": string(body)})
	}

	return nil
}

var client pmclient.Client

type testcase struct {
	testFunc  func(ctx context.Context, client pmclient.Client) error
	runConfig core.RunConfig
}

var tests = map[string]testcase{
	"hello-http-server": {
		runConfig: core.RunConfig{
			Name:    fun.Valid("http-hello-server"),
			Command: "/home/rprtr258/.gvm/gos/go1.19.5/bin/go",
			Args:    []string{"run", "tests/hello-http/main.go"},
			Tags:    nil,
			Cwd:     "",
		},
		testFunc: func(ctx context.Context, client pmclient.Client) error {
			time.Sleep(3 * time.Second)
			if errHTTP := HTTPResponse(ctx, "http://localhost:8080/", "hello world"); errHTTP != nil {
				return errHTTP
			}

			return nil
		},
	},
}

func runTest(ctx context.Context, name string, test testcase) (ererer error) {
	log.Infof("running test", log.F{"test": name})
	defer func() {
		if ererer == nil {
			log.Infof("test succeeded", log.F{"test": name})
		} else {
			log.Errorf("test failed", log.F{"test": name, "err": ererer})
		}
	}()

	ids, errCreate := client.Create(ctx, []*api.ProcessOptions{{
		Name:    test.runConfig.Name.Ptr(),
		Command: test.runConfig.Command,
		Args:    test.runConfig.Args,
		Cwd:     test.runConfig.Cwd,
		Tags:    test.runConfig.Tags,
	}})
	if errCreate != nil {
		return errCreate
	}

	if errStart := client.Start(ctx, ids); errStart != nil {
		return errStart
	}

	if errTest := test.testFunc(ctx, client); errTest != nil {
		return errTest
	}

	if _, errStop := client.Stop(ctx, ids); errStop != nil {
		return errStop
	}

	if errDelete := client.Delete(ctx, ids); errDelete != nil {
		return errDelete
	}

	return nil
}

func main() {
	testCommands := fun.ToSlice(tests, func(name string, test testcase) *cli.Command {
		return &cli.Command{
			Name: name,
			Action: func(ctx *cli.Context) error {
				return runTest(ctx.Context, name, test)
			},
		}
	})

	pmcli.App.Commands = append(pmcli.App.Commands, &cli.Command{
		Name:        "test",
		Usage:       "run e2e tests",
		Subcommands: testCommands,
		Before: func(ctx *cli.Context) error {
			var errClient error
			client, errClient = pmclient.NewGrpcClient()
			if errClient != nil {
				return errClient
			}

			_, errRestart := daemon.Restart(ctx.Context)
			if errRestart != nil {
				return errRestart
			}

			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				if client.HealthCheck(ctx.Context) == nil {
					break
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}

			return nil
		},
		After: func(ctx *cli.Context) error {
			if errKill := daemon.Kill(); errKill != nil {
				return errKill
			}

			if errHealth := client.HealthCheck(ctx.Context); errHealth == nil {
				return xerr.NewM("daemon is healthy but must not")
			}

			return nil
		},
	})

	if err := pmcli.App.Run(os.Args); err != nil {
		log.Fatal(err.Error())
	}

}
