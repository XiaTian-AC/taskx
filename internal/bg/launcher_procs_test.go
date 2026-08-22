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
	time.Sleep(3 * time.Second)
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
