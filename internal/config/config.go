package config

import (
	"os"
	"path/filepath"

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

func Shells(dir string) (map[string]string, error) {
	shells := map[string]string{}
	path := filepath.Join(dir, "config.lua")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return shells, nil
		}
		return nil, err
	}
	L := lua.NewState()
	defer L.Close()
	if err := L.DoFile(path); err != nil {
		return nil, err
	}
	if L.GetTop() == 0 {
		return shells, nil
	}
	tbl, ok := L.Get(-1).(*lua.LTable)
	if !ok {
		return shells, nil
	}
	st, ok := tbl.RawGetString("shells").(*lua.LTable)
	if !ok {
		return shells, nil
	}
	st.ForEach(func(k, v lua.LValue) {
		ks, kok := k.(lua.LString)
		vs, vok := v.(lua.LString)
		if kok && vok {
			shells[string(ks)] = string(vs)
		}
	})
	return shells, nil
}
