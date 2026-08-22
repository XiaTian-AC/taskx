package runtime

import (
	"fmt"
	"io"

	lua "github.com/yuin/gopher-lua"

	"tkx/internal/taskfile"
)

type Options struct {
	ShellName  string
	Registry   map[string]string
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
	Background bool
}

func Run(f *taskfile.File, name string, args *lua.LTable, opts Options) error {
	c := &Ctx{
		L: f.L, File: f, ShellName: opts.ShellName, Registry: opts.Registry,
		Stdout: opts.Stdout, Stderr: opts.Stderr, Stdin: opts.Stdin,
		Background: opts.Background,
	}
	task, ok := f.Tasks[name]
	if !ok {
		return fmt.Errorf("task %q not found; run 'tkx ls' to list tasks", name)
	}
	if err := callTask(f.L, c, task, args); err != nil {
		return fmt.Errorf("task %q: %v", name, err)
	}
	return nil
}
