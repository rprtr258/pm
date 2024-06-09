package linuxprocess

import (
	stdErrors "errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"time"

	"github.com/rprtr258/fun"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
)

type Stat struct {
	ShimPID        int
	Memory         uint64  // bytes
	CPU            float64 // percent
	ChildStartTime time.Time
}

// TODO: this might be called in function, call batch once instead
// TODO: this is called many times and results are not reused where they might be reused
func StatPMID(list []ProcListItem, pmid core.PMID, env string) (Stat, bool) {
	for _, p := range list {
		if p.Environ[env] != string(pmid) {
			continue
		}

		totalMemory := uint64(0)
		totalCPU := float64(0)
		children, err := p.P.Children()
		if err != nil {
			// no children, no stats
			return Stat{
				ShimPID:        p.Handle.Pid,
				Memory:         0,
				CPU:            0,
				ChildStartTime: time.Time{},
			}, true
		}

		startTimeUnix := int64(math.MaxInt64)
		for _, child := range children {
			if mem, err := child.MemoryInfo(); err == nil {
				totalMemory += mem.RSS
			}
			if cpu, err := child.CPUPercent(); err == nil {
				totalCPU = cpu
			}

			// find oldest child process
			if startUnix, err := child.CreateTime(); err == nil && startUnix < startTimeUnix {
				startTimeUnix = startUnix
			}
		}
		return Stat{
			ShimPID:        p.Handle.Pid,
			Memory:         totalMemory,
			CPU:            totalCPU,
			ChildStartTime: time.Unix(0, startTimeUnix*time.Millisecond.Nanoseconds()),
		}, true
	}
	return fun.Zero[Stat](), false
}

// Status information about the process.
// See /proc/PID/stat file struct
// e.g. https://mjmwired.net/kernel/Documentation/filesystems/proc.txt#313
type ProcessStat struct {
	Comm                string
	State               string
	Pid                 int
	Ppid                int
	Pgrp                int64
	Session             int64
	TtyNr               int64
	Tpgid               int64
	Flags               uint64
	Minflt              uint64
	Cminflt             uint64
	Majflt              uint64
	Cmajflt             uint64
	Utime               uint64
	Stime               uint64
	Cutime              int64
	Cstime              int64
	Priority            int64
	Nice                int64
	NumThreads          int64
	Itrealvalue         int64
	Starttime           uint64
	Vsize               uint64
	Rss                 int64
	Rsslim              uint64
	Startcode           uint64
	Endcode             uint64
	Startstack          uint64
	Kstkesp             uint64
	Kstkeip             uint64
	Signal              uint64
	Blocked             uint64
	Sigignore           uint64
	Sigcatch            uint64
	Wchan               uint64
	Nswap               uint64
	Cnswap              uint64
	ExitSignal          int64
	Processor           int64
	RtPriority          uint64
	Policy              uint64
	DelayacctBlkioTicks uint64
	GuestTime           uint64
	CguestTime          int64
	StartData           uint64
	EndData             uint64
	StartBrk            uint64
	ArgStart            uint64
	ArgEnd              uint64
	EnvStart            uint64
	EnvEnd              uint64
	ExitCode            int64
}

var ErrStatFileNotFound = stdErrors.New("stat file not found")

func ReadProcessStat(pid int) (ProcessStat, error) {
	statFile, err := os.Open(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		if stdErrors.Is(err, fs.ErrNotExist) {
			return fun.Zero[ProcessStat](), ErrStatFileNotFound
		}

		return fun.Zero[ProcessStat](), errors.Wrapf(err, "read proc stat file")
	}
	defer statFile.Close()

	var stat ProcessStat
	if _, err := fmt.Fscanf(
		statFile,
		"%d %s %s %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d\n", //nolint:lll // aboba
		&stat.Pid,
		&stat.Comm,
		&stat.State,
		&stat.Ppid,
		&stat.Pgrp,
		&stat.Session,
		&stat.TtyNr,
		&stat.Tpgid,
		&stat.Flags,
		&stat.Minflt,
		&stat.Cminflt,
		&stat.Majflt,
		&stat.Cmajflt,
		&stat.Utime,
		&stat.Stime,
		&stat.Cutime,
		&stat.Cstime,
		&stat.Priority,
		&stat.Nice,
		&stat.NumThreads,
		&stat.Itrealvalue,
		&stat.Starttime,
		&stat.Vsize,
		&stat.Rss,
		&stat.Rsslim,
		&stat.Startcode,
		&stat.Endcode,
		&stat.Startstack,
		&stat.Kstkesp,
		&stat.Kstkeip,
		&stat.Signal,
		&stat.Blocked,
		&stat.Sigignore,
		&stat.Sigcatch,
		&stat.Wchan,
		&stat.Nswap,
		&stat.Cnswap,
		&stat.ExitSignal,
		&stat.Processor,
		&stat.RtPriority,
		&stat.Policy,
		&stat.DelayacctBlkioTicks,
		&stat.GuestTime,
		&stat.CguestTime,
		&stat.StartData,
		&stat.EndData,
		&stat.StartBrk,
		&stat.ArgStart,
		&stat.ArgEnd,
		&stat.EnvStart,
		&stat.EnvEnd,
		&stat.ExitCode,
	); err != nil {
		return fun.Zero[ProcessStat](), errors.Wrapf(err, "read proc stat file")
	}

	return stat, nil
}
