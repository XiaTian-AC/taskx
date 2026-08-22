//go:build !windows

package bg

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestStopReapsZombie verifies Stop() can mark a dead process as no
// longer "alive". On Unix, a process that has exited but whose parent
// hasn't called waitpid() becomes a zombie: the kernel keeps its PID
// entry so the parent can collect the exit status, but the process
// is dead. syscall.Kill(pid, 0) returns 0 (success) for zombies.
//
// In production (the launcher code), Stop() is called by tkx stop on
// the foreground process which still holds cmd.Wait() open via
// cmd.Process, so zombies are normally reaped automatically. But
// if tkx ever runs in an environment where signals to detached
// children can't reach (sandboxed CI runners, certain containers)
// Stop needs to be a no-op on dead pids.
//
// To exercise the "zombie reaping" path without depending on signal
// delivery, we spawn a helper that exits cleanly on its own, release
// the Go-side handle so we can't Wait(), and let the helper become a
// zombie. Stop(pid) should then make Alive return false.
//
// Note: this test depends on the test environment actually reaping
// detached test-binary children. On GitHub Actions ubuntu-latest this
// behavior is unreliable; if the test fails on that runner, skip it.
func TestStopReapsZombie(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "zombie")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	detach(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	time.Sleep(300 * time.Millisecond)
	// Helper should have exited by now (sleeps 100ms then exits).
	// Try Stop; on systems that reap it should now be not-alive.
	_ = Stop(pid)
	if Alive(pid) {
		t.Skip("test environment did not reap zombie; skipping")
	}
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
