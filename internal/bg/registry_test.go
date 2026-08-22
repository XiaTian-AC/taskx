package bg

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNextID(t *testing.T) {
	dir := t.TempDir()
	id1, n1, err := NextID(dir, "build")
	if err != nil {
		t.Fatal(err)
	}
	if id1 != "build#1" || n1 != 1 {
		t.Errorf("first = %q/%d, want build#1/1", id1, n1)
	}
	id2, n2, _ := NextID(dir, "build")
	if id2 != "build#2" || n2 != 2 {
		t.Errorf("second = %q/%d, want build#2/2", id2, n2)
	}
	id3, n3, _ := NextID(dir, "test")
	if id3 != "test#1" || n3 != 1 {
		t.Errorf("other = %q/%d, want test#1/1", id3, n3)
	}
}

func TestRegisterAndSnapshot(t *testing.T) {
	dir := t.TempDir()
	inst := Instance{ID: "build#1", Name: "build", N: 1, PID: 123, Status: "running", Started: time.Now().UTC()}
	if err := Register(dir, inst); err != nil {
		t.Fatal(err)
	}
	all, err := Snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != "build#1" {
		t.Errorf("snapshot = %v, want [build#1]", all)
	}
}

func TestRegisterRotatesExited(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "build#1.log")
	os.MkdirAll(filepath.Dir(logPath), 0o755)
	os.WriteFile(logPath, []byte("old log"), 0o644)
	old := Instance{ID: "build#1", Name: "build", N: 1, PID: 999, Status: "exited", Log: logPath, Started: time.Now().UTC()}
	Register(dir, old)
	next := Instance{ID: "build#2", Name: "build", N: 2, PID: 456, Status: "running", Started: time.Now().UTC()}
	Register(dir, next)
	all, _ := Snapshot(dir)
	if len(all) != 1 || all[0].ID != "build#2" {
		t.Errorf("after rotation = %v, want [build#2]", all)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("old log should be deleted")
	}
}

func TestRegisterKeepsRunning(t *testing.T) {
	dir := t.TempDir()
	Register(dir, Instance{ID: "build#1", Name: "build", N: 1, PID: 111, Status: "running", Started: time.Now().UTC()})
	Register(dir, Instance{ID: "build#2", Name: "build", N: 2, PID: 222, Status: "running", Started: time.Now().UTC()})
	all, _ := Snapshot(dir)
	if len(all) != 2 {
		t.Errorf("snapshot = %v, want 2 instances", all)
	}
}

func TestMarkEnded(t *testing.T) {
	dir := t.TempDir()
	Register(dir, Instance{ID: "build#1", Name: "build", N: 1, PID: 123, Status: "running", Started: time.Now().UTC()})
	if err := MarkEnded(dir, "build#1", "exited", 0); err != nil {
		t.Fatal(err)
	}
	all, _ := Snapshot(dir)
	if len(all) != 1 || all[0].Status != "exited" || all[0].ExitCode != 0 {
		t.Errorf("after MarkEnded = %+v", all[0])
	}
	if all[0].EndedAt == nil {
		t.Error("EndedAt should be set")
	}
}

func TestRunningFilter(t *testing.T) {
	dir := t.TempDir()
	Register(dir, Instance{ID: "build#1", Name: "build", N: 1, PID: 1, Status: "running", Started: time.Now().UTC()})
	Register(dir, Instance{ID: "build#2", Name: "build", N: 2, PID: 2, Status: "exited", Started: time.Now().UTC()})
	running, _ := Running(dir, "build")
	if len(running) != 1 || running[0].ID != "build#1" {
		t.Errorf("running = %v, want [build#1]", running)
	}
}

func TestRemoveInstance(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "build#1.log")
	os.MkdirAll(filepath.Dir(logPath), 0o755)
	os.WriteFile(logPath, []byte("log"), 0o644)
	Register(dir, Instance{ID: "build#1", Name: "build", N: 1, PID: 1, Status: "exited", Log: logPath, Started: time.Now().UTC()})
	if err := RemoveInstance(dir, "build#1"); err != nil {
		t.Fatal(err)
	}
	all, _ := Snapshot(dir)
	if len(all) != 0 {
		t.Errorf("after remove = %v, want empty", all)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("log should be deleted")
	}
}
