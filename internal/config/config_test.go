package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "taskx")
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestDataDirXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	got, err := DataDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "taskx")
	if got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
}

func TestEnsureDataDirCreates(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir, err := EnsureDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(filepath.Join(dir, "logs")); err != nil || !st.IsDir() {
		t.Errorf("logs subdir not created: %v", err)
	}
}

func TestShellsMissing(t *testing.T) {
	shells, err := Shells(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(shells) != 0 {
		t.Errorf("expected empty, got %v", shells)
	}
}

func TestShellsLoaded(t *testing.T) {
	dir := t.TempDir()
	content := `return { shells = { bash = "/usr/bin/bash", pwsh = "/usr/bin/pwsh" } }`
	os.WriteFile(filepath.Join(dir, "config.lua"), []byte(content), 0o644)
	shells, err := Shells(dir)
	if err != nil {
		t.Fatal(err)
	}
	if shells["bash"] != "/usr/bin/bash" || shells["pwsh"] != "/usr/bin/pwsh" {
		t.Errorf("unexpected shells: %v", shells)
	}
}

func TestShellsMalformed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.lua"), []byte(`this is not lua!!!`), 0o644)
	_, err := Shells(dir)
	if err == nil {
		t.Error("expected error for malformed config.lua")
	}
}
