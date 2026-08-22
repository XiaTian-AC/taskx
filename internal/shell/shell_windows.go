//go:build windows

package shell

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	execLookPath = exec.LookPath
	whereGit     = func() (string, error) {
		out, err := exec.Command("where.exe", "git").Output()
		if err != nil {
			return "", err
		}
		sc := bufio.NewScanner(strings.NewReader(string(out)))
		if sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line != "" {
				return line, nil
			}
		}
		return "", errors.New("git not found")
	}
	fileExists = func(p string) bool {
		st, err := os.Stat(p)
		return err == nil && !st.IsDir()
	}
)

func resolveBuiltin(name string) (Spec, error) {
	switch name {
	case "pwsh":
		for _, cand := range []string{"pwsh.exe", "pwsh", "powershell.exe"} {
			if p, err := execLookPath(cand); err == nil {
				return Spec{Path: p, Args: argsFor("pwsh")}, nil
			}
		}
		return Spec{}, fmt.Errorf("shell %q: pwsh/powershell not found in PATH", name)
	case "bash":
		if p, err := findGitBash(); err == nil {
			return Spec{Path: p, Args: argsFor("bash")}, nil
		} else {
			return Spec{}, fmt.Errorf("shell %q: %w (register in config.lua: shells = { bash = \"...\" })", name, err)
		}
	case "sh":
		if p, err := execLookPath("sh.exe"); err == nil {
			return Spec{Path: p, Args: argsFor("sh")}, nil
		}
		return Spec{}, fmt.Errorf("shell %q: sh.exe not found in PATH", name)
	}
	return Spec{}, fmt.Errorf("unknown shell %q (register in config.lua under shells)", name)
}

func findGitBash() (string, error) {
	if git, err := whereGit(); err == nil {
		root := filepath.Dir(filepath.Dir(git))
		p := filepath.Join(root, "bin", "bash.exe")
		if fileExists(p) {
			return p, nil
		}
	}
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin", "bash.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "bin", "bash.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Git", "bin", "bash.exe"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "scoop", "apps", "git", "current", "bin", "bash.exe"),
		)
	}
	if scoop := os.Getenv("SCOOP"); scoop != "" {
		candidates = append(candidates, filepath.Join(scoop, "apps", "git", "current", "bin", "bash.exe"))
	}
	for _, p := range candidates {
		if fileExists(p) {
			return p, nil
		}
	}
	return "", errors.New("git bash not found")
}
