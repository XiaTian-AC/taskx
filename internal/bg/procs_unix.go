//go:build !windows

package bg

import (
	"syscall"
	"time"
)

func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func Stop(pid int) error {
	if !Alive(pid) {
		return nil
	}
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		pgid = pid
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	for i := 0; i < 10; i++ {
		if !Alive(pid) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	for i := 0; i < 20; i++ {
		if !Alive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Last resort: reap zombie if process exists but signals don't reach
	// it (e.g. sandboxed CI runners where session-group signal delivery
	// is blocked). Try waitpid(WNOHANG); ignore errors.
	if Alive(pid) {
		var status syscall.WaitStatus
		_, _, werr := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
		_ = werr // ESRCH is fine (already reaped); other errors ignored
	}
	return nil
}
