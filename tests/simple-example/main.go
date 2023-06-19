package main

import (
	"fmt"
	"net/http"
	"os"
)

var response = []byte(os.Getenv("RESPONSE"))

func handle(w http.ResponseWriter, req *http.Request) {
	if _, err := w.Write(response); err != nil {
		panic(err)
	}
}

func main() {
	host, okHost := os.LookupEnv("HOST")
	if !okHost {
		host = "0.0.0.0"
	}

	port := os.Getenv("PORT")
	listenAddress := host + ":" + port
	fmt.Println("listening on:", listenAddress)

	http.HandleFunc("/", handle)
	if err := http.ListenAndServe(listenAddress, nil); err != nil { //nolint:gosec // example program
		panic(err)
	}
}
