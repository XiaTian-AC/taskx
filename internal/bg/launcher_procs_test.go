//go:build !windows

package bg

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestStopKillsProcess verifies Stop() can actually kill a long-running
// process tree. We use /bin/sleep because it responds to SIGTERM and
// is available everywhere on Unix. (The previous test-helper pattern
// doesn't work in GitHub Actions ubuntu-latest sandboxed runners:
// detached test-binary children become zombies that are never reaped,
// so Alive(pid, 0) keeps returning true after the process actually
// exited. Stop() itself works fine in production against real shells,
// which is covered by integration_test.go.)
func TestStopKillsProcess(t *testing.T) {
	cmd := exec.Command("sleep", "300")
	detach(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	time.Sleep(100 * time.Millisecond)
	if !Alive(pid) {
		t.Fatal("expected sleep to be alive")
	}
	if err := Stop(pid); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("sleep still alive after Stop")
}

// TestStopDeadPidNoop verifies Stop on a non-existent pid returns nil
// without erroring.
func TestStopDeadPidNoop(t *testing.T) {
	if err := Stop(999999); err != nil {
		t.Fatalf("Stop(999999) = %v, want nil", err)
	}
}

// keep imports referenced
var _ = os.Getenv
var _ = testing.Short
var _ time.Duration

func TestBuildRunArgv(t *testing.T) {
	got := buildRunArgv("build", "bash", "build#3", []string{"--tag", "v1"})
	want := []string{"_run", "build", "--shell", "bash", "--id", "build#3", "--", "--tag", "v1"}
	if len(got) != len(want) {
		t.Fatalf("buildRunArgv = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildRunArgvNoShell(t *testing.T) {
	got := buildRunArgv("test", "", "test#1", nil)
	want := []string{"_run", "test", "--id", "test#1", "--"}
	if len(got) != len(want) {
		t.Fatalf("buildRunArgv = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
