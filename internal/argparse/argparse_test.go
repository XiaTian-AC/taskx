package argparse

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestParseHeuristic(t *testing.T) {
	p, err := Parse([]string{"--tag", "v0.1.2", "--force", "file.txt"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags["tag"] != "v0.1.2" {
		t.Errorf("tag = %v", p.Flags["tag"])
	}
	if p.Flags["force"] != "file.txt" {
		t.Errorf("force = %v, want file.txt (heuristic consumes next non-dash token)", p.Flags["force"])
	}
	if len(p.Positional) != 0 {
		t.Errorf("Positional = %v, want empty", p.Positional)
	}
}

func TestParseHeuristicBoolThenSeparator(t *testing.T) {
	p, err := Parse([]string{"--tag", "v0.1.2", "--force", "--", "file.txt"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags["force"] != true {
		t.Errorf("force = %v, want true", p.Flags["force"])
	}
	if len(p.Positional) != 1 || p.Positional[0] != "file.txt" {
		t.Errorf("Positional = %v, want [file.txt]", p.Positional)
	}
}

func TestParseEqualsForm(t *testing.T) {
	p, err := Parse([]string{"--tag=v0.2.0"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags["tag"] != "v0.2.0" {
		t.Errorf("tag = %v", p.Flags["tag"])
	}
}

func TestParseDoubleDashSeparator(t *testing.T) {
	p, err := Parse([]string{"--force", "--", "--tag", "x"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags["force"] != true {
		t.Errorf("force = %v", p.Flags["force"])
	}
	if len(p.Positional) != 2 || p.Positional[0] != "--tag" || p.Positional[1] != "x" {
		t.Errorf("Positional = %v", p.Positional)
	}
}

func TestParseStrictStringFlag(t *testing.T) {
	specs := map[string]Spec{"tag": {Type: "string"}}
	p, err := Parse([]string{"--tag", "v1"}, specs)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags["tag"] != "v1" {
		t.Errorf("tag = %v", p.Flags["tag"])
	}
}

func TestParseStrictStringFlagMissingValue(t *testing.T) {
	specs := map[string]Spec{"tag": {Type: "string"}}
	_, err := Parse([]string{"--tag"}, specs)
	if err == nil || !strings.Contains(err.Error(), "requires a value") {
		t.Errorf("expected 'requires a value' error, got %v", err)
	}
}

func TestParseStrictBoolFlag(t *testing.T) {
	specs := map[string]Spec{"force": {Type: "bool"}}
	p, err := Parse([]string{"--force"}, specs)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags["force"] != true {
		t.Errorf("force = %v", p.Flags["force"])
	}
}

func TestParseStrictBoolFlagEquals(t *testing.T) {
	specs := map[string]Spec{"force": {Type: "bool"}}
	p, err := Parse([]string{"--force=false"}, specs)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags["force"] != false {
		t.Errorf("force = %v", p.Flags["force"])
	}
}

func TestParseStrictUnknownFlag(t *testing.T) {
	specs := map[string]Spec{"tag": {Type: "string"}}
	_, err := Parse([]string{"--unknown"}, specs)
	if err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("expected 'unknown flag' error, got %v", err)
	}
	if !strings.Contains(err.Error(), "allowed") || !strings.Contains(err.Error(), "--tag") {
		t.Errorf("error should list allowed flags, got %q", err.Error())
	}
}

func TestParseRequiredMissing(t *testing.T) {
	specs := map[string]Spec{"tag": {Type: "string", Required: true, Desc: "version tag"}}
	_, err := Parse([]string{}, specs)
	if err == nil || !strings.Contains(err.Error(), "missing required") {
		t.Errorf("expected 'missing required' error, got %v", err)
	}
}

func TestToLuaTable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	p := &Parsed{
		Flags:      map[string]any{"tag": "v1", "force": true},
		Positional: []string{"a", "b"},
	}
	tbl := ToLuaTable(L, p)
	if tbl.RawGetString("tag").String() != "v1" {
		t.Errorf("tag = %v", tbl.RawGetString("tag"))
	}
	if tbl.RawGetString("force") != lua.LTrue {
		t.Errorf("force = %v", tbl.RawGetString("force"))
	}
	pos, ok := tbl.RawGetString("_").(*lua.LTable)
	if !ok {
		t.Fatal("positional _ is not a table")
	}
	if pos.RawGetInt(1).String() != "a" || pos.RawGetInt(2).String() != "b" {
		t.Errorf("positional = %v %v", pos.RawGetInt(1), pos.RawGetInt(2))
	}
}
