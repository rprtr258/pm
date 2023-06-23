package main

import (
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/rprtr258/log"
	"golang.org/x/exp/slog"
)

const _addr = ":8080"

func handler(w http.ResponseWriter, r *http.Request) {
	dump, errDump := httputil.DumpRequest(r, true)
	if errDump != nil {
		slog.Error("dump request error", "err", errDump)
		os.Exit(1)
	}

	slog.Info("request", "dump", string(dump))

	if _, err := w.Write([]byte("hello world")); err != nil {
		slog.Error("write response", "err", err.Error())
	}
}

func main() {
	slog.SetDefault(slog.New(log.New()))

	slog.Info("listening", "addr", _addr)

	errListen := http.ListenAndServe(_addr, http.HandlerFunc(handler)) //nolint:gosec // acceptable for example
	slog.Error(errListen.Error())
	os.Exit(1)
}
