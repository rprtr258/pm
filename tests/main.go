package main

import (
	"log"
	"net/http"

	"github.com/davecgh/go-spew/spew"
)

const _addr = ":8080"

func main() {
	log.Printf("listening on %s\n", _addr)
	log.Fatal(http.ListenAndServe(_addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spew.Dump(r)
		_, err := w.Write([]byte("hello world"))
		if err != nil {
			log.Println(err)
		}
	})))
}
