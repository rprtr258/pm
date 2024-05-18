package linuxprocess

import (
	"os"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

type ProcListItem struct {
	Handle  *os.Process
	P       *process.Process
	Environ map[string]string
}

func List() []ProcListItem {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	procs := make([]ProcListItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			continue
		}

		pp, err := process.NewProcess(int32(proc.Pid))
		if err != nil {
			continue
		}

		environKVs, err := pp.Environ()
		if err != nil {
			continue
		}

		environ := map[string]string{}
		for _, kv := range environKVs {
			kv := strings.SplitN(kv, "=", 2)
			if len(kv) != 2 {
				// NOTE: for some fucking reason there might be empty key-value line
				continue
			}

			environ[kv[0]] = kv[1]
		}

		procs = append(procs, ProcListItem{
			Handle:  proc,
			P:       pp,
			Environ: environ,
		})
	}
	return procs
}
