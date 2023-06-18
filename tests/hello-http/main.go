package main

import (
	"net/http"
	"net/http/httputil"

	"github.com/rprtr258/log"
)

const _addr = ":8080"

func handler(w http.ResponseWriter, r *http.Request) {
	dump, errDump := httputil.DumpRequest(r, true)
	if errDump != nil {
		log.Fatalf("dump request error", log.F{"err": errDump})
	}

	log.Infof("request", log.F{"dump": string(dump)})

	if _, err := w.Write([]byte("hello world")); err != nil {
		log.Errorf("write response", log.F{"err": err.Error()})
	}
}

func main() {
	log.Infof("listening", log.F{"addr": _addr})

	errListen := http.ListenAndServe(_addr, http.HandlerFunc(handler)) //nolint:gosec // acceptable for example
	log.Fatal(errListen.Error())
}
