package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/rs/zerolog/log"
)

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

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: hello-http [addr]")
	}

	addr := os.Args[1]
	log.Info().Str("addr", addr).Msg("listening")

	return http.ListenAndServe(addr, http.HandlerFunc(handler))
}

func main() {
	log.Fatal().Msg(run().Error())
}
