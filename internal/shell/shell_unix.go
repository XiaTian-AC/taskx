//go:build !windows

package shell

import (
	"fmt"
	"os/exec"
)

var execLookPath = exec.LookPath

func resolveBuiltin(name string) (Spec, error) {
	switch name {
	case "pwsh", "bash", "sh":
		p, err := execLookPath(name)
		if err != nil {
			return Spec{}, fmt.Errorf("shell %q: not found in PATH (register in config.lua: shells = { %q = \"/abs/path\" })", name, name)
		}
		return Spec{Path: p, Args: argsFor(name)}, nil
	}
	return Spec{}, fmt.Errorf("unknown shell %q (register in config.lua under shells)", name)
}
