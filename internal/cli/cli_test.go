package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	bg.Register(dataDir, bg.Instance{
		ID: "build#1", Name: "build", N: 1, PID: 99999, Status: "exited", ExitCode: 0,
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
