package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tkx/internal/argparse"
	"tkx/internal/bg"
	"tkx/internal/config"
)

func setupEnv(t *testing.T, taskfileContent string) (Deps, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("XDG_DATA_HOME", dataDir)
	if taskfileContent != "" {
		tfDir := filepath.Join(cfgDir, "taskx")
		os.MkdirAll(tfDir, 0o755)
		os.WriteFile(filepath.Join(tfDir, "Taskfile.lua"), []byte(taskfileContent), 0o644)
	}
	var out, errOut bytes.Buffer
	return Deps{
		SelfPath: os.Args[0],
		Version:  "test",
		Stdin:    strings.NewReader(""),
		Stdout:   &out,
		Stderr:   &errOut,
	}, &out, &errOut
}

func TestRunNoArgs(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	code := Run(nil, d)
	if code != 2 {
		t.Errorf("no args: code = %d, want 2", code)
	}
	if !strings.Contains(out.String(), "Usage") {
		t.Errorf("no args: should print usage")
	}
}

func TestRunVersion(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	code := Run([]string{"version"}, d)
	if code != 0 || !strings.Contains(out.String(), "test") {
		t.Errorf("version: code=%d out=%q", code, out.String())
	}
}

func TestRunLs(t *testing.T) {
	d, out, _ := setupEnv(t, `return {
  build = function(ctx) ctx:sh("cargo build") end,
  test = { desc = "run tests", run = function(ctx) ctx:sh("cargo test") end },
}`)
	code := Run([]string{"ls"}, d)
	if code != 0 {
		t.Errorf("ls: code = %d", code)
	}
	s := out.String()
	if !strings.Contains(s, "build") || !strings.Contains(s, "test") || !strings.Contains(s, "run tests") {
		t.Errorf("ls output = %q", s)
	}
}

func TestRunTask(t *testing.T) {
	d, out, _ := setupEnv(t, `return {
  hello = function(ctx) ctx:echo("task ran") end,
}`)
	code := Run([]string{"hello"}, d)
	if code != 0 {
		t.Errorf("hello: code = %d", code)
	}
	if !strings.Contains(out.String(), "task ran") {
		t.Errorf("hello output = %q", out.String())
	}
}

func TestRunTaskWithArgs(t *testing.T) {
	d, out, _ := setupEnv(t, `return {
  echo = {
    args = { msg = { type = "string", required = true } },
    run = function(ctx, args) ctx:echo("got=" .. args.msg) end,
  },
}`)
	code := Run([]string{"echo", "--msg", "hello"}, d)
	if code != 0 {
		t.Errorf("echo: code = %d", code)
	}
	if !strings.Contains(out.String(), "got=hello") {
		t.Errorf("echo output = %q", out.String())
	}
}

func TestRunTaskNotFound(t *testing.T) {
	d, _, errOut := setupEnv(t, `return { a = function(ctx) end }`)
	code := Run([]string{"nope"}, d)
	if code != 1 || !strings.Contains(errOut.String(), "not found") {
		t.Errorf("nope: code=%d err=%q", code, errOut.String())
	}
}

func TestHelpTask(t *testing.T) {
	d, out, _ := setupEnv(t, `return {
  release = {
    desc = "release the project",
    args = {
      tag = { type = "string", required = true, desc = "version tag" },
    },
    run = function(ctx, args) end,
  },
}`)
	code := Run([]string{"help", "release"}, d)
	if code != 0 {
		t.Errorf("help release: code = %d", code)
	}
	s := out.String()
	if !strings.Contains(s, "release the project") || !strings.Contains(s, "--tag") {
		t.Errorf("help release output = %q", s)
	}
}

func TestLsRunningEmpty(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	code := Run([]string{"ls-running"}, d)
	if code != 0 || !strings.Contains(out.String(), "no background") {
		t.Errorf("ls-running empty: code=%d out=%q", code, out.String())
	}
}

func TestLsRunningWithInstance(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	dataDir, _ := config.DataDir()
	now := time.Now().UTC()
	bg.Register(dataDir, bg.Instance{
		ID: "build#1", Name: "build", N: 1, PID: 99999, Status: "exited", ExitCode: 0,
		Started: now, EndedAt: &now,
	})
	code := Run([]string{"ls-running"}, d)
	if code != 0 || !strings.Contains(out.String(), "build#1") {
		t.Errorf("ls-running: code=%d out=%q", code, out.String())
	}
}

func TestExtractShell(t *testing.T) {
	name, rest := extractShell([]string{"--tag", "v1", "--shell", "bash", "extra"})
	if name != "bash" || len(rest) != 3 || rest[0] != "--tag" || rest[1] != "v1" || rest[2] != "extra" {
		t.Errorf("extractShell: name=%q rest=%v", name, rest)
	}
}

func TestExtractShellEquals(t *testing.T) {
	name, rest := extractShell([]string{"--shell=bash", "pos"})
	if name != "bash" || len(rest) != 1 || rest[0] != "pos" {
		t.Errorf("extractShell=: name=%q rest=%v", name, rest)
	}
}

func TestExtractShellNotAfterDoubleDash(t *testing.T) {
	name, rest := extractShell([]string{"--", "--shell", "bash"})
	if name != "" || len(rest) != 3 {
		t.Errorf("extractShell should not cross --: name=%q rest=%v", name, rest)
	}
}

func TestConfirmYes(t *testing.T) {
	ok := confirm("test?", strings.NewReader("y\n"), &bytes.Buffer{})
	if !ok {
		t.Error("confirm('y') should return true")
	}
}

func TestConfirmNo(t *testing.T) {
	ok := confirm("test?", strings.NewReader("n\n"), &bytes.Buffer{})
	if ok {
		t.Error("confirm('n') should return false")
	}
}

func TestFindInstanceExact(t *testing.T) {
	all := []bg.Instance{
		{ID: "build#1", Name: "build", N: 1},
		{ID: "build#2", Name: "build", N: 2},
	}
	inst, err := findInstance(all, "build#1")
	if err != nil || inst.ID != "build#1" {
		t.Errorf("findInstance exact: %v %v", inst, err)
	}
}

func TestFindInstanceNewest(t *testing.T) {
	all := []bg.Instance{
		{ID: "build#1", Name: "build", N: 1},
		{ID: "build#2", Name: "build", N: 2},
	}
	inst, err := findInstance(all, "build")
	if err != nil || inst.ID != "build#2" {
		t.Errorf("findInstance newest: %v %v", inst, err)
	}
}

func TestFindInstanceMissing(t *testing.T) {
	_, err := findInstance(nil, "nope")
	if err == nil {
		t.Error("expected error for missing instance")
	}
}
func TestLsRunningFilterOld(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	dataDir, _ := config.DataDir()
	started := time.Now().UTC().Add(-2 * time.Hour)
	ended := time.Now().UTC().Add(-2 * time.Hour)
	bg.Register(dataDir, bg.Instance{
		ID: "build#1", Name: "build", N: 1, PID: 99999, Status: "exited",
		Started: started, EndedAt: &ended,
	})
	code := Run([]string{"ls-running"}, d)
	if code != 0 || !strings.Contains(out.String(), "no recent") {
		t.Errorf("expected filtered: code=%d out=%q", code, out.String())
	}
}

func TestLsRunningRunningFirst(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	dataDir, _ := config.DataDir()
	now := time.Now().UTC()
	bg.Register(dataDir, bg.Instance{ID: "old-task#1", Name: "old-task", N: 1, PID: 1, Status: "exited", Started: now, EndedAt: &now})
	bg.Register(dataDir, bg.Instance{ID: "live-task#1", Name: "live-task", N: 1, PID: 99999, Status: "running", Started: now})
	code := Run([]string{"ls-running"}, d)
	if code != 0 {
		t.Fatal(code)
	}
	s := out.String()
	idxRun := strings.Index(s, "live-task#1")
	idxExit := strings.Index(s, "old-task#1")
	if idxRun < 0 || idxExit < 0 || idxRun > idxExit {
		t.Errorf("running should come first: %q", s)
	}
}

func TestCleanAll(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	dataDir, _ := config.DataDir()
	now := time.Now().UTC()
	bg.Register(dataDir, bg.Instance{ID: "stale#1", Name: "stale", N: 1, PID: 1, Status: "exited", Started: now, EndedAt: &now})
	bg.Register(dataDir, bg.Instance{ID: "live#1", Name: "live", N: 1, PID: 99999, Status: "running", Started: now})
	code := Run([]string{"clean"}, d)
	if code != 0 {
		t.Fatalf("clean exit = %d, out=%q", code, out.String())
	}
	all, _ := bg.Snapshot(dataDir)
	if len(all) != 1 || all[0].ID != "live#1" {
		t.Errorf("after clean = %v, want only running", all)
	}
	if !strings.Contains(out.String(), "cleaned 1") {
		t.Errorf("output = %q, want 'cleaned 1'", out.String())
	}
}

func TestCleanOlderThan(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	dataDir, _ := config.DataDir()
	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour)
	recent := now.Add(-10 * time.Minute)
	bg.Register(dataDir, bg.Instance{ID: "build#1", Name: "build", N: 1, PID: 1, Status: "exited", Started: old, EndedAt: &old})
	bg.Register(dataDir, bg.Instance{ID: "build#2", Name: "build", N: 2, PID: 2, Status: "exited", Started: now, EndedAt: &recent})
	code := Run([]string{"clean", "--older-than=1h"}, d)
	if code != 0 {
		t.Fatalf("clean exit = %d, out=%q", code, out.String())
	}
	all, _ := bg.Snapshot(dataDir)
	if len(all) != 1 || all[0].ID != "build#2" {
		t.Errorf("after clean older-than = %v, want only recent", all)
	}
}

func TestCleanOlderThanRunningSkipped(t *testing.T) {
	d, _, _ := setupEnv(t, "")
	dataDir, _ := config.DataDir()
	now := time.Now().UTC()
	bg.Register(dataDir, bg.Instance{ID: "build#1", Name: "build", N: 1, PID: 1, Status: "running", Started: now})
	code := Run([]string{"clean"}, d)
	if code != 0 {
		t.Fatalf("clean exit = %d", code)
	}
	all, _ := bg.Snapshot(dataDir)
	if len(all) != 1 {
		t.Errorf("running instance should be kept")
	}
}
func TestArgsHint(t *testing.T) {
	if got := argsHint(nil); got != "" {
		t.Errorf("argsHint(nil) = %q, want empty", got)
	}
	specs := map[string]argparse.Spec{
		"tag":   {Type: "string", Required: true},
		"force": {Type: "bool"},
	}
	got := argsHint(specs)
	want := "[args: --force, --tag (required)]"
	if got != want {
		t.Errorf("argsHint = %q, want %q", got, want)
	}
}

func TestLsShowsRequiredArgs(t *testing.T) {
	d, out, _ := setupEnv(t, `return {
  release = {
    desc = "release it",
    args = { tag = { type = "string", required = true } },
    run = function(ctx, args) end,
  },
  plain = function(ctx) end,
}`)
	code := Run([]string{"ls"}, d)
	if code != 0 {
		t.Fatalf("ls exit = %d", code)
	}
	s := out.String()
	if !strings.Contains(s, "--tag (required)") {
		t.Errorf("ls output should mark required arg: %q", s)
	}
	if strings.Contains(s, "plain  ") && strings.Contains(strings.Split(s, "\n")[strings.Index(s, "plain")], "[args") {
		t.Errorf("task without args should not have args hint")
	}
}

func TestExtractTaskfile(t *testing.T) {
	path, rest := extractTaskfile([]string{"--taskfile", "/tmp/x.lua", "hello"})
	if path != "/tmp/x.lua" || len(rest) != 1 || rest[0] != "hello" {
		t.Errorf("extractTaskfile: path=%q rest=%v", path, rest)
	}
	path, rest = extractTaskfile([]string{"--taskfile=/y.lua"})
	if path != "/y.lua" || len(rest) != 0 {
		t.Errorf("extractTaskfile=: path=%q rest=%v", path, rest)
	}
	path, rest = extractTaskfile([]string{"--", "--taskfile", "x.lua"})
	if path != "" || len(rest) != 3 {
		t.Errorf("extractTaskfile should stop at --: path=%q rest=%v", path, rest)
	}
}

func TestRunWithTaskfileOverride(t *testing.T) {
	d, out, errOut := setupEnv(t, `return {
  global_only = function(ctx) ctx:echo("from-global") end,
}`)
	localDir := t.TempDir()
	localTF := filepath.Join(localDir, "My.lua")
	os.WriteFile(localTF, []byte(`return {
  local_task = function(ctx) ctx:echo("from-local") end,
}`), 0o644)

	code := Run([]string{"--taskfile", localTF, "local_task"}, d)
	if code != 0 || !strings.Contains(out.String(), "from-local") {
		t.Errorf("--taskfile run: code=%d out=%q", code, out.String())
	}

	out.Reset()
	errOut.Reset()
	code = Run([]string{"--taskfile=" + localTF, "global_only"}, d)
	if code == 0 || !strings.Contains(errOut.String(), "not found") {
		t.Errorf("override should hide global tasks: code=%d err=%q", code, errOut.String())
	}
}

func TestConfigCommand(t *testing.T) {
	d, out, _ := setupEnv(t, "")
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	tfDir := filepath.Join(cfgDir, "taskx")
	os.MkdirAll(tfDir, 0o755)
	os.WriteFile(filepath.Join(tfDir, "config.lua"), []byte(`return {
  shells = { gitbash = "C:/bin/bash.exe" },
  display = { ls_running = { time = "2d", running_first = false } },
}`), 0o644)

	code := Run([]string{"config"}, d)
	if code != 0 {
		t.Fatalf("config exit = %d", code)
	}
	s := out.String()
	if !strings.Contains(s, "time          = 2d") || !strings.Contains(s, "running_first = false") {
		t.Errorf("display section wrong:\n%s", s)
	}
	if !strings.Contains(s, "gitbash") {
		t.Errorf("shells section missing gitbash:\n%s", s)
	}

	out.Reset()
	code = Run([]string{"config", "shells"}, d)
	if code != 0 || !strings.Contains(out.String(), "gitbash") || strings.Contains(out.String(), "[display") {
		t.Errorf("config shells filter: code=%d out=%q", code, out.String())
	}

	out.Reset()
	code = Run([]string{"config", "bogus"}, d)
	if code != 2 {
		t.Errorf("unknown section should exit 2, got %d", code)
	}
}
