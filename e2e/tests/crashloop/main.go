package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("trying to wake up")
	time.Sleep(time.Second)
	fmt.Println("nah, going back to sleep")
	os.Exit(1)
}
