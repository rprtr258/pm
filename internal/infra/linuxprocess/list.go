package linuxprocess

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

type ProcListItem struct {
	Handle  *os.Process
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

		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
		if err != nil {
			continue
		}

		environ := map[string]string{}
		for len(b) > 0 {
			eqAt := bytes.IndexByte(b, '=')
			if eqAt == -1 {
				break // TODO: ???
			}

			sepAt := bytes.IndexByte(b, 0)
			if sepAt < eqAt {
				break // TODO: ???
			}

			k := b[:eqAt]
			v := b[eqAt+1 : sepAt]
			environ[string(k)] = string(v)
			if sepAt == len(b)-1 {
				break
			}
			b = b[sepAt+1:]
		}

		procs = append(procs, ProcListItem{
			Handle:  proc,
			Environ: environ,
		})
	}
	return procs
}
