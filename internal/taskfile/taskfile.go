package taskfile

import (
	"fmt"
	"os"
	"regexp"
	"sort"

	lua "github.com/yuin/gopher-lua"
)

var nameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

type ArgSpec struct {
	Type     string
	Required bool
	Desc     string
}

type Task struct {
	Name string
	Desc string
	Fn   *lua.LFunction
	Args map[string]ArgSpec
}

type File struct {
	L     *lua.LState
	Tasks map[string]*Task
	Order []string
}

func Load(path string) (*File, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no Taskfile at %s (create it with your tasks)", path)
		}
		return nil, err
	}
	L := lua.NewState()
	if err := L.DoFile(path); err != nil {
		L.Close()
		return nil, fmt.Errorf("Taskfile.lua: %w", err)
	}
	if L.GetTop() == 0 || L.Get(-1).Type() != lua.LTTable {
		L.Close()
		return nil, fmt.Errorf("Taskfile.lua must return a tasks table")
	}
	tbl := L.Get(-1).(*lua.LTable)
	f := &File{L: L, Tasks: map[string]*Task{}}
	tbl.ForEach(func(k, v lua.LValue) {
		key, ok := k.(lua.LString)
		if !ok {
			return
		}
		name := string(key)
		if !nameRe.MatchString(name) {
			fmt.Fprintf(os.Stderr, "tkx: warning: skipping task %q (invalid name)\n", name)
			return
		}
		switch tv := v.(type) {
		case *lua.LFunction:
			f.Tasks[name] = &Task{Name: name, Fn: tv}
			f.Order = append(f.Order, name)
		case *lua.LTable:
			run, ok := tv.RawGetString("run").(*lua.LFunction)
			if !ok {
				fmt.Fprintf(os.Stderr, "tkx: warning: skipping task %q (table form needs a run function)\n", name)
				return
			}
			task := &Task{Name: name, Fn: run}
			if d, ok := tv.RawGetString("desc").(lua.LString); ok {
				task.Desc = string(d)
			}
			if at, ok := tv.RawGetString("args").(*lua.LTable); ok {
				task.Args = map[string]ArgSpec{}
				at.ForEach(func(ak, av lua.LValue) {
					an, ok := ak.(lua.LString)
					if !ok {
						return
					}
					sp := ArgSpec{Type: "string"}
					if spt, ok := av.(*lua.LTable); ok {
						if ty, ok := spt.RawGetString("type").(lua.LString); ok {
							sp.Type = string(ty)
						}
						if rq, ok := spt.RawGetString("required").(lua.LBool); ok {
							sp.Required = bool(rq)
						}
						if ds, ok := spt.RawGetString("desc").(lua.LString); ok {
							sp.Desc = string(ds)
						}
					}
					task.Args[string(an)] = sp
				})
			}
			f.Tasks[name] = task
			f.Order = append(f.Order, name)
		default:
			fmt.Fprintf(os.Stderr, "tkx: warning: skipping task %q (must be function or table)\n", name)
		}
	})
	if len(f.Order) == 0 {
		L.Close()
		return nil, fmt.Errorf("Taskfile.lua defines no tasks")
	}
	sort.Strings(f.Order)
	return f, nil
}

func (f *File) Close() { f.L.Close() }

func (f *File) Get(name string) (*Task, bool) {
	t, ok := f.Tasks[name]
	return t, ok
}
