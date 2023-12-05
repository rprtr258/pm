package linuxprocess

import (
	"os"
	"strings"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
)

type Status struct {
	Args          []string
	Envs          map[string]string
	Executable    string
	Cwd           string
	Groups        []int
	PageSize      int
	Hostname      string
	UserCacheDir  string
	UserConfigDir string
	UserHomeDir   string
	PID           int
	PPID          int
	PGID          int
	PGRP          int
	UID           int
	EUID          int
	GID           int
	EGID          int
	TID           int
}

func GetSelfStatus() (Status, error) {
	executable, err := os.Executable()
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "get executable")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "get cwd")
	}

	groups, err := os.Getgroups()
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "get groups")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "get hostname")
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "get userCacheDir")
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "get userConfigDir")
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "get userHomeDir")
	}

	pgid, err := syscall.Getpgid(syscall.Getpid())
	if err != nil {
		return fun.Zero[Status](), xerr.NewWM(err, "get pgid")
	}

	return Status{
		Args: os.Args,
		Envs: fun.SliceToMap[string, string](func(v string) (string, string) {
			name, val, _ := strings.Cut(v, "=")
			return name, val
		}, syscall.Environ()...),
		Executable:    executable,
		Cwd:           cwd,
		Groups:        groups,
		PageSize:      syscall.Getpagesize(),
		Hostname:      hostname,
		UserCacheDir:  userCacheDir,
		UserConfigDir: userConfigDir,
		UserHomeDir:   userHomeDir,
		PID:           syscall.Getpid(),
		PGID:          pgid,
		PPID:          syscall.Getppid(),
		PGRP:          syscall.Getpgrp(),
		UID:           syscall.Getuid(),
		EUID:          syscall.Geteuid(),
		GID:           syscall.Getgid(),
		EGID:          syscall.Getegid(),
		TID:           syscall.Gettid(),
	}, nil
}
