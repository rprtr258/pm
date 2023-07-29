package daemon

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestCompilation(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("short mode")
	}

	if !requireMinor(5) {
		t.Skip(runtime.Version(), "cross-compilation requires compiler bootstrapping")
	}

	env := os.Environ()

	for _, platform := range []struct {
		goos, goarch string
	}{
		{"darwin", "amd64"},
		{"dragonfly", "amd64"},
		{"freebsd", "386"},
		{"freebsd", "amd64"},
		{"freebsd", "arm"},
		{"linux", "386"},
		{"linux", "amd64"},
		{"linux", "arm"},
		{"linux", "arm64"},
		{"netbsd", "386"},
		{"netbsd", "amd64"},
		{"netbsd", "arm"},
		{"openbsd", "386"},
		{"openbsd", "amd64"},
		{"openbsd", "arm"},
		{"solaris", "amd64"},
		{"windows", "386"},
		{"windows", "amd64"},
	} {
		platform := platform
		t.Run(platform.goos+"/"+platform.goarch, func(t *testing.T) {
			t.Parallel()

			if platform.goos == "solaris" && !requireMinor(7) {
				t.Log("skip, solaris requires at least go1.7")
				return
			}

			cmd := exec.Command("go", "build", "./")
			cmd.Env = append(append([]string(nil), env...), "GOOS="+platform.goos, "GOARCH="+platform.goarch)
			out, err := cmd.CombinedOutput()
			if len(out) > 0 {
				t.Log(platform, "\n", string(out))
			}
			if err != nil {
				t.Error(platform, err)
			}
		})
	}
}

func requireMinor(minor int) bool {
	str := runtime.Version()
	if !strings.HasPrefix(str, "go1.") {
		return true
	}

	str = strings.TrimPrefix(str, "go1.")
	ver, err := strconv.Atoi(str)
	if err != nil {
		return false
	}

	return ver >= minor
}
