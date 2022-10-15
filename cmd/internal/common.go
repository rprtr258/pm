package internal

import (
	"os"
	"path"
)

// Processes - proc name -> pid
var (
	Processes map[string]int = make(map[string]int)
	UserHome                 = os.Getenv("HOME")
	HomeDir                  = path.Join(UserHome, ".pm")
)
