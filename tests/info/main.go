package main

import (
	"fmt"
	"os"
	"sort"
)

func printErr[T any](name string, f func() (T, error)) {
	if value, err := f(); err != nil {
		fmt.Printf("Error getting %s: %s\n", name, err.Error())
	} else {
		fmt.Printf("%s: %v\n", name, value)
	}
}

func main() {
	fmt.Println("Args:", os.Args)

	fmt.Println("Envs:")
	environ := os.Environ()
	sort.Strings(environ)
	for _, env := range environ {
		fmt.Println("\t", env)
	}

	printErr("Executable", os.Executable)

	printErr("CWD", os.Getwd)

	printErr("Groups", os.Getgroups)
	fmt.Println("page size:", os.Getpagesize())
	printErr("hostname", os.Hostname)
	printErr("user cache dir", os.UserCacheDir)
	printErr("user config dir", os.UserConfigDir)
	printErr("user home dir", os.UserHomeDir)

	fmt.Println("PID  (process ID)        :", os.Getpid())
	fmt.Println("PPID (parent process ID) :", os.Getppid())
	fmt.Println("UID  (user ID)           :", os.Getuid())
	fmt.Println("EUID (effective user ID) :", os.Geteuid())
	fmt.Println("GID  (group ID)          :", os.Getgid())
	fmt.Println("EGID (effective group ID):", os.Getegid())
}
