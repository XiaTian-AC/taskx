//go:build windows

package bg

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(10 * time.Second)
	os.Exit(0)
}

func TestAliveAndStop(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	detach(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	time.Sleep(200 * time.Millisecond)
	if !Alive(pid) {
		t.Fatal("expected process to be alive")
	}
	if err := Stop(pid); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("process still alive after Stop")
}