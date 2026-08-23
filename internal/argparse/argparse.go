package argparse

import (
	"fmt"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type Parsed struct {
	Flags      map[string]any
	Positional []string
}

type Spec struct {
	Type     string
	Required bool
	Desc     string
}

func Parse(argv []string, specs map[string]Spec) (*Parsed, error) {
	p := &Parsed{Flags: map[string]any{}}
	i := 0
	for i < len(argv) {
		a := argv[i]
		switch {
		case a == "--":
			p.Positional = append(p.Positional, argv[i+1:]...)
			return p, validate(p, specs)
		case strings.HasPrefix(a, "--") && len(a) > 2:
			body := a[2:]
			if eq := strings.IndexByte(body, '='); eq >= 0 {
				name, val := body[:eq], body[eq+1:]
				if err := setFlag(p, specs, name, val); err != nil {
					return nil, err
				}
				i++
				continue
			}
			sp, hasSpec := specs[body]
			if hasSpec && sp.Type == "bool" {
				p.Flags[body] = true
				i++
				continue
			}
			if hasSpec && sp.Type == "string" {
				if i+1 >= len(argv) {
					return nil, fmt.Errorf("flag --%s requires a value", body)
				}
				p.Flags[body] = argv[i+1]
				i += 2
				continue
			}
			if !hasSpec && specs != nil {
				allowed := make([]string, 0, len(specs))
				for k := range specs {
					allowed = append(allowed, "--"+k)
				}
				sort.Strings(allowed)
				return nil, fmt.Errorf("unknown flag --%s (allowed: %s)", body, strings.Join(allowed, ", "))
			}
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
				p.Flags[body] = argv[i+1]
				i += 2
			} else {
				p.Flags[body] = true
				i++
			}
		default:
			p.Positional = append(p.Positional, a)
			i++
		}
	}
	return p, validate(p, specs)
}

func setFlag(p *Parsed, specs map[string]Spec, name, val string) error {
	sp, hasSpec := specs[name]
	if specs != nil && !hasSpec {
		return fmt.Errorf("unknown flag --%s", name)
	}
	if hasSpec && sp.Type == "bool" {
		switch val {
		case "true":
			p.Flags[name] = true
		case "false":
			p.Flags[name] = false
		default:
			return fmt.Errorf("flag --%s: want bool (true/false), got %q", name, val)
		}
		return nil
	}
	p.Flags[name] = val
	return nil
}

func validate(p *Parsed, specs map[string]Spec) error {
	if specs == nil {
		return nil
	}
	for name, sp := range specs {
		if sp.Required {
			if _, ok := p.Flags[name]; !ok {
				return fmt.Errorf("missing required flag --%s: %s", name, sp.Desc)
			}
		}
	}
	return nil
}

func ToLuaTable(L *lua.LState, p *Parsed) *lua.LTable {
	t := L.NewTable()
	for k, v := range p.Flags {
		switch vv := v.(type) {
		case string:
			t.RawSetString(k, lua.LString(vv))
		case bool:
			t.RawSetString(k, lua.LBool(vv))
		}
	}
	pos := L.NewTable()
	for i, s := range p.Positional {
		pos.RawSetInt(i+1, lua.LString(s))
	}
	t.RawSetString("_", pos)
	return t
}
