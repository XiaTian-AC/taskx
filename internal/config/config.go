package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	lua "github.com/yuin/gopher-lua"
)

func ConfigDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "taskx"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "taskx"), nil
}

func DataDir() (string, error) {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "taskx"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "taskx"), nil
}

func EnsureDataDir() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func TaskfilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Taskfile.lua"), nil
}

// Display holds display-related config options.
type Display struct {
	LsRunning LsRunningDisplay
}

type LsRunningDisplay struct {
	Time         time.Duration // 0 = running only; otherwise window for ended instances
	RunningFirst bool          // running instances on top
	NewestFirst  bool          // newest first within each group
}

// DefaultDisplay returns the default display config.
func DefaultDisplay() Display {
	return Display{
		LsRunning: LsRunningDisplay{
			Time:         time.Hour,
			RunningFirst: true,
			NewestFirst:  true,
		},
	}
}

// Load reads config.lua from dir and returns the parsed options.
// Missing file returns defaults with no error.
func Load(dir string) (Display, map[string]string, error) {
	display := DefaultDisplay()
	shells := map[string]string{}
	path := filepath.Join(dir, "config.lua")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return display, shells, nil
		}
		return display, shells, err
	}
	L := lua.NewState()
	defer L.Close()
	if err := L.DoFile(path); err != nil {
		return display, shells, err
	}
	if L.GetTop() == 0 {
		return display, shells, nil
	}
	tbl, ok := L.Get(-1).(*lua.LTable)
	if !ok {
		return display, shells, fmt.Errorf("config.lua: expected table, got %s", L.Get(-1).Type().String())
	}

	if st, ok := tbl.RawGetString("shells").(*lua.LTable); ok {
		st.ForEach(func(k, v lua.LValue) {
			ks, kok := k.(lua.LString)
			vs, vok := v.(lua.LString)
			if kok && vok {
				shells[string(ks)] = string(vs)
			}
		})
	} else if tbl.RawGetString("shells") != lua.LNil {
		return display, shells, fmt.Errorf("config.lua: 'shells' must be a table, got %s", tbl.RawGetString("shells").Type().String())
	}

	if disp, ok := tbl.RawGetString("display").(*lua.LTable); ok {
		if lr, ok := disp.RawGetString("ls_running").(*lua.LTable); ok {
			if s, ok := lr.RawGetString("time").(lua.LString); ok {
				d, err := ParseDuration(string(s))
				if err != nil {
					return display, shells, fmt.Errorf("config.lua: display.ls_running.time: %w", err)
				}
				display.LsRunning.Time = d
			}
			if v, ok := lr.RawGetString("running_first").(lua.LBool); ok {
				display.LsRunning.RunningFirst = bool(v)
			}
			if v, ok := lr.RawGetString("newest_first").(lua.LBool); ok {
				display.LsRunning.NewestFirst = bool(v)
			}
		}
	}

	return display, shells, nil
}

// Shells loads only the shells registry. Convenience wrapper around Load.
func Shells(dir string) (map[string]string, error) {
	_, shells, err := Load(dir)
	return shells, err
}

// ParseDuration parses a short duration string:
//   "0"           -> 0 (running only)
//   "30" or "30h" -> 30 hours (h is default unit)
//   "30m"         -> 30 minutes
//   "2d"          -> 2 days
//   "1w"          -> 1 week
func ParseDuration(s string) (time.Duration, error) {
	if s == "" || s == "0" {
		return 0, nil
	}
	if len(s) > 1 && s[0] >= '0' && s[0] <= '9' {
		suffix := s[len(s)-1]
		if suffix >= '0' && suffix <= '9' {
			s = s + "h"
		}
	}
	var n int
	var unit byte
	if _, err := fmt.Sscanf(s, "%d%c", &n, &unit); err != nil {
		if err.Error() == "unexpected EOF" || err == fmt.Errorf("missing verb") {
			if _, err2 := fmt.Sscanf(s, "%d", &n); err2 == nil {
				return time.Duration(n) * time.Hour, nil
			}
		}
		return 0, fmt.Errorf("invalid duration %q (use 0, 30m, 1h, 2d, 1w)", s)
	}
	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("invalid duration unit %q (use m/h/d/w)", string(unit))
}
