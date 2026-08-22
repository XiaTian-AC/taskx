package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"0", 0},
		{"", 0},
		{"30m", 30 * time.Minute},
		{"1h", time.Hour},
		{"2d", 48 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
		{"30", 30 * time.Hour},
	}
	for _, c := range cases {
		got, err := ParseDuration(c.in)
		if err != nil {
			t.Errorf("ParseDuration(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseDuration(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseDurationInvalid(t *testing.T) {
	for _, in := range []string{"abc", "1x", "h"} {
		if _, err := ParseDuration(in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestLoadDisplayDefaults(t *testing.T) {
	dir := t.TempDir()
	disp, shells, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if disp.LsRunning.Time != time.Hour {
		t.Errorf("default time = %v, want 1h", disp.LsRunning.Time)
	}
	if !disp.LsRunning.RunningFirst || !disp.LsRunning.NewestFirst {
		t.Error("default sort flags should be true")
	}
	if len(shells) != 0 {
		t.Errorf("default shells = %v, want empty", shells)
	}
}

func TestLoadDisplayCustom(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.lua"), []byte(`return {
  shells = { bash = "/x/bash" },
  display = {
    ls_running = {
      time = "2d",
      running_first = false,
      newest_first = true,
    },
  },
}`), 0o644)
	disp, shells, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if disp.LsRunning.Time != 48*time.Hour {
		t.Errorf("custom time = %v, want 2d", disp.LsRunning.Time)
	}
	if disp.LsRunning.RunningFirst {
		t.Error("custom running_first should be false")
	}
	if shells["bash"] != "/x/bash" {
		t.Errorf("custom shells = %v", shells)
	}
}

func TestLoadDisplayInvalidDuration(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.lua"), []byte(`return {
  display = { ls_running = { time = "1x" } },
}`), 0o644)
	_, _, err := Load(dir)
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}
