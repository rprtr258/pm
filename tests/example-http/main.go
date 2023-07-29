package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
)

var response = []byte(os.Getenv("RESPONSE"))

func handle(w http.ResponseWriter, _ *http.Request) {
	if _, err := w.Write(response); err != nil {
		panic(err)
	}
}

func handleEcho(w http.ResponseWriter, r *http.Request) {
	request, err := httputil.DumpRequest(r, true)
	if err != nil {
		panic(err)
	}

	if _, err := w.Write(request); err != nil {
		panic(err)
	}
}

func main() {
	host, okHost := os.LookupEnv("HOST")
	if !okHost {
		host = "0.0.0.0"
	}

	port, okPort := os.LookupEnv("PORT")
	if !okPort {
		port = "8080"
	}

	listenAddress := host + ":" + port
	fmt.Println("listening on:", listenAddress)

	http.HandleFunc("/", handle)
	http.HandleFunc("/echo", handleEcho)
	if err := http.ListenAndServe(listenAddress, nil); err != nil { //nolint:gosec // example program
		panic(err)
	}
}
