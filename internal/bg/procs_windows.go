//go:build windows

package bg

import (
	"fmt"
	"os/exec"
	"strings"
)

func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), fmt.Sprintf(`"%d"`, pid))
}

func Stop(pid int) error {
	if !Alive(pid) {
		return nil
	}
	return exec.Command("taskkill", "/PID", fmt.Sprint(pid), "/T", "/F").Run()
}
