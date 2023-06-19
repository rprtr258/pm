package main

import (
	"log"
	"os"
	"strconv"
)

func main() {
	exitCode := 0
	if len(os.Args) > 1 {
		var err error
		exitCode, err = strconv.Atoi(os.Args[0])
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	os.Exit(exitCode)
}
