package shell

import "testing"

func TestArgsFor(t *testing.T) {
	cases := []struct {
		name string
		want []string
	}{
		{"pwsh", []string{"-NoProfile", "-Command"}},
		{"bash", []string{"-euo", "pipefail", "-c"}},
		{"sh", []string{"-e", "-c"}},
		{"custom", []string{"-c"}},
	}
	for _, c := range cases {
		got := argsFor(c.name)
		if len(got) != len(c.want) {
			t.Errorf("argsFor(%q) = %v, want %v", c.name, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("argsFor(%q)[%d] = %q, want %q", c.name, i, got[i], c.want[i])
			}
		}
	}
}

func TestSpecArgv(t *testing.T) {
	s := Spec{Path: "/bin/bash", Args: []string{"-euo", "pipefail", "-c"}}
	got := s.Argv("echo hi")
	want := []string{"-euo", "pipefail", "-c", "echo hi"}
	if len(got) != len(want) {
		t.Fatalf("Argv = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("Argv[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveRegistryOverride(t *testing.T) {
	reg := map[string]string{"bash": "/custom/bash"}
	s, err := Resolve("bash", reg)
	if err != nil {
		t.Fatal(err)
	}
	if s.Path != "/custom/bash" {
		t.Errorf("Path = %q, want /custom/bash", s.Path)
	}
	if len(s.Args) != 3 || s.Args[0] != "-euo" {
		t.Errorf("Args = %v, want bash args", s.Args)
	}
}

func TestResolveUnknownShell(t *testing.T) {
	_, err := Resolve("noshell", map[string]string{})
	if err == nil {
		t.Error("expected error for unknown shell")
	}
}
