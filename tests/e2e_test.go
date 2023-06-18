package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core/daemon"
	"github.com/rprtr258/pm/pkg/client"
)

func startDaemon(ctx context.Context, t *testing.T, client client.Client) { //nolint:thelper // not helper
	_, errRestart := daemon.Restart(ctx)
	assert.NoError(t, errRestart)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if client.HealthCheck(ctx) == nil {
			break
		}

		select {
		case <-ctx.Done():
			assert.Fail(t, ctx.Err().Error())
		case <-ticker.C:
		}
	}
}

func stopDaemon(ctx context.Context, t *testing.T, client client.Client) { //nolint:thelper // not helper
	assert.NoError(t, daemon.Kill())
	assert.Error(t, client.HealthCheck(ctx))
}

func Test_e2e(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, errClient := client.NewGrpcClient()
	assert.NoError(t, errClient)

	startDaemon(ctx, t, client)

	for name, _ := range map[string]struct{}{} {
		t.Run(name, func(t *testing.T) { // TODO: run in separate docker container
			_, errCreate := client.Create(ctx, []*api.ProcessOptions{})
			assert.NoError(t, errCreate)
		})
	}

	stopDaemon(ctx, t, client)
}
