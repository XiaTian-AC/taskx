package taskfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempTaskfile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Taskfile.lua")
	os.WriteFile(path, []byte(content), 0o644)
	return path
}

func TestLoadFunctionForm(t *testing.T) {
	path := writeTempTaskfile(t, `return {
  build = function(ctx) ctx:sh("cargo build") end,
}`)
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	task, ok := f.Get("build")
	if !ok {
		t.Fatal("build task not found")
	}
	if task.Desc != "" {
		t.Errorf("Desc = %q, want empty", task.Desc)
	}
	if task.Args != nil {
		t.Errorf("Args = %v, want nil for function form", task.Args)
	}
}

func TestLoadTableForm(t *testing.T) {
	path := writeTempTaskfile(t, `return {
  release = {
    desc = "test, commit, push",
    args = {
      tag   = { type = "string", required = true, desc = "git tag" },
      force = { type = "bool", required = false },
    },
    run = function(ctx, args) ctx:echo(args.tag) end,
  },
}`)
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	task, ok := f.Get("release")
	if !ok {
		t.Fatal("release task not found")
	}
	if task.Desc != "test, commit, push" {
		t.Errorf("Desc = %q", task.Desc)
	}
	if len(task.Args) != 2 {
		t.Fatalf("Args = %v, want 2 entries", task.Args)
	}
	tag := task.Args["tag"]
	if tag.Type != "string" || !tag.Required || tag.Desc != "git tag" {
		t.Errorf("tag spec = %+v", tag)
	}
	force := task.Args["force"]
	if force.Type != "bool" || force.Required {
		t.Errorf("force spec = %+v", force)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.lua"))
	if err == nil || !strings.Contains(err.Error(), "no Taskfile") {
		t.Errorf("expected 'no Taskfile' error, got %v", err)
	}
}

func TestLoadSyntaxError(t *testing.T) {
	path := writeTempTaskfile(t, `this is not valid lua!!!`)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for syntax error")
	}
}

func TestLoadNoReturn(t *testing.T) {
	path := writeTempTaskfile(t, `local x = 1`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "must return") {
		t.Errorf("expected 'must return' error, got %v", err)
	}
}

func TestLoadEmpty(t *testing.T) {
	path := writeTempTaskfile(t, `return {}`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "no tasks") {
		t.Errorf("expected 'no tasks' error, got %v", err)
	}
}

func TestLoadOrderSorted(t *testing.T) {
	path := writeTempTaskfile(t, `return {
  zebra = function(ctx) end,
  alpha = function(ctx) end,
  mango = function(ctx) end,
}`)
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	want := []string{"alpha", "mango", "zebra"}
	if len(f.Order) != len(want) {
		t.Fatalf("Order = %v, want %v", f.Order, want)
	}
	for i, name := range want {
		if f.Order[i] != name {
			t.Errorf("Order[%d] = %q, want %q", i, f.Order[i], name)
		}
	}
}
