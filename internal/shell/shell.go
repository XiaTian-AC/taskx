package shell

import (
	"runtime"
)

type Spec struct {
	Path string
	Args []string
}

func (s Spec) Argv(cmd string) []string {
	out := make([]string, 0, len(s.Args)+1)
	out = append(out, s.Args...)
	out = append(out, cmd)
	return out
}

func argsFor(name string) []string {
	switch name {
	case "pwsh":
		return []string{"-NoProfile", "-Command"}
	case "bash":
		return []string{"-euo", "pipefail", "-c"}
	case "sh":
		return []string{"-e", "-c"}
	default:
		return []string{"-c"}
	}
}

func Resolve(name string, registry map[string]string) (Spec, error) {
	if name == "" {
		if runtime.GOOS == "windows" {
			name = "pwsh"
		} else {
			name = "bash"
		}
	}
	if path, ok := registry[name]; ok {
		return Spec{Path: path, Args: argsFor(name)}, nil
	}
	return resolveBuiltin(name)
}
