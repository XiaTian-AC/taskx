package bg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWatchFollowsAppends(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	os.WriteFile(logPath, []byte("line1\n"), 0o644)

	var out bytes.Buffer
	alive := true
	done := make(chan struct{})
	go func() {
		Watch(logPath, "test#1", func() bool { return alive }, &out)
		close(done)
	}()

	time.Sleep(300 * time.Millisecond)
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("line2\n")
	f.Close()
	time.Sleep(300 * time.Millisecond)
	alive = false
	<-done

	s := out.String()
	if !strings.Contains(s, "line1") {
		t.Errorf("missing line1: %q", s)
	}
	if !strings.Contains(s, "line2") {
		t.Errorf("missing line2: %q", s)
	}
	if !strings.Contains(s, "exited") {
		t.Errorf("missing exit marker: %q", s)
	}
}

func TestWatchTruncation(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	os.WriteFile(logPath, []byte("old content here\n"), 0o644)

	var out bytes.Buffer
	alive := true
	done := make(chan struct{})
	go func() {
		Watch(logPath, "test#1", func() bool { return alive }, &out)
		close(done)
	}()

	time.Sleep(300 * time.Millisecond)
	os.WriteFile(logPath, []byte("new\n"), 0o644)
	time.Sleep(300 * time.Millisecond)
	alive = false
	<-done

	if !strings.Contains(out.String(), "new") {
		t.Errorf("missing new content after truncation: %q", out.String())
	}
}

func TestWatchMissingFile(t *testing.T) {
	err := Watch("/nonexistent/path.log", "test#1", nil, &bytes.Buffer{})
	if err == nil {
		t.Error("expected error for missing log file")
	}
}
