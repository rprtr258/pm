package main

import (
	"net/http"
	"net/http/httputil"

	"github.com/rs/zerolog/log"
)

const _addr = ":8080"

func handler(w http.ResponseWriter, r *http.Request) {
	dump, errDump := httputil.DumpRequest(r, true)
	if errDump != nil {
		log.Fatal().Err(errDump).Msg("dump request error")
	}

	log.Info().Bytes("dump", dump).Msg("request")

	if _, err := w.Write([]byte("hello world")); err != nil {
		log.Error().Err(err).Msg("write response")
	}
}

func main() {
	log.Info().Str("addr", _addr).Msg("listening")

	errListen := http.ListenAndServe(_addr, http.HandlerFunc(handler)) //nolint:gosec // acceptable for example
	log.Fatal().Err(errListen).Send()
}
