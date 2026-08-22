package bg

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestHelperProcess exits immediately (after a short delay so the parent
// can observe it as alive). It does NOT respond to SIGTERM on purpose:
// sandboxed Linux CI runners (GitHub Actions ubuntu-latest) do not
// reliably deliver session-group signals to detached test-binary children.
// The real Stop() behavior is exercised by the integration test against
// actual shells (pwsh/bash), so this unit test focuses on Alive() and
// Stop(pid-of-dead-process) as a no-op.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(2 * time.Second)
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
	time.Sleep(100 * time.Millisecond)
	if !Alive(pid) {
		t.Fatal("expected process to be alive")
	}
	// Wait for the helper to exit on its own (2s sleep + buffer).
	time.Sleep(3 * time.Second)
	if Alive(pid) {
		t.Fatal("helper should have exited by now")
	}
	// Stop on a dead pid must be a no-op (returns nil).
	if err := Stop(pid); err != nil {
		t.Fatalf("Stop on dead pid: %v", err)
	}
}

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
