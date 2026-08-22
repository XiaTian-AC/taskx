package runtime

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tkx/internal/taskfile"
)

func writeTaskfile(t *testing.T, content string) *taskfile.File {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "Taskfile.lua")
	os.WriteFile(path, []byte(content), 0o644)
	f, err := taskfile.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.Close)
	return f
}

func runTask(t *testing.T, f *taskfile.File, name string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(f, name, f.L.NewTable(), Options{
		Stdout: &out, Stderr: &out, Stdin: os.Stdin,
		Registry: map[string]string{},
	})
	return out.String(), err
}

func TestRunEcho(t *testing.T) {
	f := writeTaskfile(t, `return {
  hello = function(ctx) ctx:echo("hello", "world") end,
}`)
	out, err := runTask(t, f, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("output = %q, want 'hello world'", out)
	}
}

func TestRunSh(t *testing.T) {
	f := writeTaskfile(t, `return {
  shell = function(ctx) ctx:sh("echo from-shell") end,
}`)
	out, err := runTask(t, f, "shell")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "from-shell") {
		t.Errorf("output = %q, want 'from-shell'", out)
	}
}

func TestRunShFail(t *testing.T) {
	f := writeTaskfile(t, `return {
  fail = function(ctx) ctx:sh("exit 3") end,
}`)
	_, err := runTask(t, f, "fail")
	if err == nil {
		t.Fatal("expected error for failing command")
	}
	if !strings.Contains(err.Error(), "exit 3") {
		t.Errorf("error = %v, want exit code 3", err)
	}
}

func TestRunChain(t *testing.T) {
	f := writeTaskfile(t, `return {
  base = function(ctx) ctx:echo("base-ran") end,
  chain = function(ctx) ctx:run("base") ctx:echo("chained") end,
}`)
	out, err := runTask(t, f, "chain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "base-ran") || !strings.Contains(out, "chained") {
		t.Errorf("output = %q, want base-ran + chained", out)
	}
}

func TestRunEnvOs(t *testing.T) {
	t.Setenv("TKX_TEST_ENV", "envval")
	f := writeTaskfile(t, `return {
  info = function(ctx)
    ctx:echo("env=" .. tostring(ctx:env("TKX_TEST_ENV")))
    ctx:echo("os=" .. tostring(ctx:os()))
  end,
}`)
	out, err := runTask(t, f, "info")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "env=envval") {
		t.Errorf("output = %q, want env=envval", out)
	}
}

func TestRunNotFound(t *testing.T) {
	f := writeTaskfile(t, `return { a = function(ctx) ctx:echo("a") end }`)
	err := Run(f, "nope", f.L.NewTable(), Options{
		Stdout: io.Discard, Stderr: io.Discard, Stdin: os.Stdin,
		Registry: map[string]string{},
	})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}
