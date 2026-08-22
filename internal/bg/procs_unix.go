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
	for i := 0; i < 50; i++ {
		if !Alive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	return nil
}
