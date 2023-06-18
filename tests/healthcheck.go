package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	t *testing.T,
	endpoint, expectedResponse string,
) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	assert.NoError(t, err, "failed to create request")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err, "failed to get response from %q", endpoint)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "failed to read response body")

	body = bytes.TrimSpace(body)
	assert.Equal(t, expectedResponse, string(body))
}
