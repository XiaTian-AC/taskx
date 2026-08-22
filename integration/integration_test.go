//go:build integration

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	dir, err := os.MkdirTemp("", "tkx-integration-")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "tkx.test"+suffix)
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = projectRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func runBin(t *testing.T, bin string, args ...string) (string, int) {
	t.Helper()
	var out bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	return out.String(), code
}

func TestIntegrationFlow(t *testing.T) {
	bin := buildBinary(t)
	cfgDir, _ := os.MkdirTemp("", "tkx-cfg-")
	dataDir, _ := os.MkdirTemp("", "tkx-data-")
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("XDG_DATA_HOME", dataDir)

	taskfile := `return {
  echo = function(ctx) ctx:echo("hello from echo task") end,
  quick = function(ctx)
    ctx:sh("echo line1")
    ctx:sh("echo line2")
    ctx:sh("echo line3")
  end,
  longrun = function(ctx)
    ctx:sh("echo will-sleep")
    if ctx:os() == "windows" then
      ctx:sh("Start-Sleep -Seconds 30")
    else
      ctx:sh("sleep 30")
    end
  end,
  tagged = {
    desc = "echo the tag",
    args = { tag = { type = "string", required = true, desc = "a tag" } },
    run = function(ctx, args) ctx:echo("tag=" .. args.tag) end,
  },
}`
	tfDir := filepath.Join(cfgDir, "taskx")
	os.MkdirAll(tfDir, 0o755)
	os.WriteFile(filepath.Join(tfDir, "Taskfile.lua"), []byte(taskfile), 0o644)

	out, code := runBin(t, bin, "ls")
	if code != 0 || !strings.Contains(out, "echo") || !strings.Contains(out, "tagged") {
		t.Fatalf("ls: code=%d out=%q", code, out)
	}

	out, code = runBin(t, bin, "echo")
	if code != 0 || !strings.Contains(out, "hello from echo task") {
		t.Fatalf("echo: code=%d out=%q", code, out)
	}

	out, code = runBin(t, bin, "tagged", "--tag", "v1.0")
	if code != 0 || !strings.Contains(out, "tag=v1.0") {
		t.Fatalf("tagged: code=%d out=%q", code, out)
	}

	out, code = runBin(t, bin, "bstart", "quick")
	if code != 0 || !strings.Contains(out, "started quick#1") {
		t.Fatalf("bstart quick: code=%d out=%q", code, out)
	}

	out, _ = runBin(t, bin, "ls-running")
	if !strings.Contains(out, "quick#1") {
		t.Fatalf("ls-running: %q", out)
	}

	deadline := time.Now().Add(30 * time.Second)
	finished := false
	for time.Now().Before(deadline) {
		out, _ = runBin(t, bin, "ls-running")
		if strings.Contains(out, "exited") {
			finished = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !finished {
		t.Fatal("quick task did not finish within 30s")
	}

	out, code = runBin(t, bin, "watch", "quick#1")
	if code != 0 {
		t.Fatalf("watch: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "line1") || !strings.Contains(out, "line3") || !strings.Contains(out, "exited") {
		t.Fatalf("watch output missing content: %q", out)
	}

	out, _ = runBin(t, bin, "bstart", "longrun")
	if !strings.Contains(out, "started longrun#1") {
		t.Fatalf("bstart longrun: %q", out)
	}
	time.Sleep(1 * time.Second)

	out, code = runBin(t, bin, "stop", "longrun#1")
	if code != 0 || !strings.Contains(out, "stopped") {
		t.Fatalf("stop: code=%d out=%q", code, out)
	}

	out, _ = runBin(t, bin, "ls-running")
	if !strings.Contains(out, "longrun#1") {
		t.Fatalf("ls-running after stop: %q", out)
	}

	out, _ = runBin(t, bin, "bstart", "quick")
	if !strings.Contains(out, "started quick#2") {
		t.Fatalf("bstart quick #2: %q", out)
	}
	logsDir := filepath.Join(dataDir, "taskx", "logs")
	entries, _ := os.ReadDir(logsDir)
	for _, f := range entries {
		if strings.Contains(f.Name(), "quick#1") {
			t.Errorf("old quick#1 log should be rotated, found: %s", f.Name())
		}
	}
}
