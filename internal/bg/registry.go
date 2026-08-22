package bg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Instance struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	N        int        `json:"n"`
	PID      int        `json:"pid"`
	Args     []string   `json:"args,omitempty"`
	CWD      string     `json:"cwd"`
	Started  time.Time  `json:"started"`
	Log      string     `json:"log"`
	Status   string     `json:"status"`
	ExitCode int        `json:"exit_code,omitempty"`
	EndedAt  *time.Time `json:"ended_at,omitempty"`
}

type registryData struct {
	Counters  map[string]int `json:"counters"`
	Instances []Instance     `json:"instances"`
}

func registryPath(dataDir string) string {
	return filepath.Join(dataDir, "run.json")
}

func withLock(lockPath string, fn func() error) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			defer os.Remove(lockPath)
			defer f.Close()
			return fn()
		}
		if st, serr := os.Stat(lockPath); serr == nil && time.Since(st.ModTime()) > 10*time.Second {
			os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return errors.New("timeout acquiring registry lock")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func loadData(path string) (registryData, error) {
	d := registryData{Counters: map[string]int{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return d, nil
		}
		return d, err
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return d, fmt.Errorf("corrupt registry %s: %w", path, err)
	}
	if d.Counters == nil {
		d.Counters = map[string]int{}
	}
	return d, nil
}

func saveData(path string, d registryData) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func NextID(dataDir, name string) (string, int, error) {
	path := registryPath(dataDir)
	var id string
	var n int
	err := withLock(path+".lock", func() error {
		d, err := loadData(path)
		if err != nil {
			return err
		}
		d.Counters[name]++
		n = d.Counters[name]
		id = fmt.Sprintf("%s#%d", name, n)
		return saveData(path, d)
	})
	return id, n, err
}

func Register(dataDir string, inst Instance) error {
	path := registryPath(dataDir)
	return withLock(path+".lock", func() error {
		d, err := loadData(path)
		if err != nil {
			return err
		}
		var kept []Instance
		for _, existing := range d.Instances {
			if existing.Name == inst.Name && existing.Status != "running" {
				if existing.Log != "" {
					os.Remove(existing.Log)
				}
				continue
			}
			kept = append(kept, existing)
		}
		kept = append(kept, inst)
		d.Instances = kept
		return saveData(path, d)
	})
}

func MarkEnded(dataDir, id, status string, code int) error {
	path := registryPath(dataDir)
	return withLock(path+".lock", func() error {
		d, err := loadData(path)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		for i := range d.Instances {
			if d.Instances[i].ID == id {
				d.Instances[i].Status = status
				d.Instances[i].ExitCode = code
				d.Instances[i].EndedAt = &now
				break
			}
		}
		return saveData(path, d)
	})
}

func Snapshot(dataDir string) ([]Instance, error) {
	d, err := loadData(registryPath(dataDir))
	if err != nil {
		return nil, err
	}
	return d.Instances, nil
}

func Running(dataDir, name string) ([]Instance, error) {
	all, err := Snapshot(dataDir)
	if err != nil {
		return nil, err
	}
	var out []Instance
	for _, inst := range all {
		if inst.Name == name && inst.Status == "running" {
			out = append(out, inst)
		}
	}
	return out, nil
}
