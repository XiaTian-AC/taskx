package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"tkx/internal/shell"
	"tkx/internal/taskfile"
)

type Ctx struct {
	L          *lua.LState
	File       *taskfile.File
	ShellName  string
	Registry   map[string]string
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
	Background bool
	reader     *bufio.Reader
	callDepth  int
	mt         *lua.LTable
}

const maxCallDepth = 100

func (c *Ctx) metatable(L *lua.LState) *lua.LTable {
	if c.mt == nil {
		mt := L.NewTable()
		mt.RawSetString("__index", mt)
		mt.RawSetString("sh", L.NewFunction(ctxSh))
		mt.RawSetString("exec", L.NewFunction(ctxExec))
		mt.RawSetString("run", L.NewFunction(ctxRun))
		mt.RawSetString("echo", L.NewFunction(ctxEcho))
		mt.RawSetString("ask", L.NewFunction(ctxAsk))
		mt.RawSetString("cwd", L.NewFunction(ctxCwd))
		mt.RawSetString("os", L.NewFunction(ctxOs))
		c.mt = mt
	}
	return c.mt
}

func newCtxUserData(L *lua.LState, c *Ctx) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = c
	L.SetMetatable(ud, c.metatable(L))
	return ud
}

func getCtx(L *lua.LState) *Ctx {
	ud := L.CheckUserData(1)
	c, ok := ud.Value.(*Ctx)
	if !ok {
		L.ArgError(1, "expected ctx")
	}
	return c
}

func ctxSh(L *lua.LState) int {
	c := getCtx(L)
	cmd := L.CheckString(2)
	shellName := c.ShellName
	interactive := false
	if L.GetTop() >= 3 {
		if opts, ok := L.Get(3).(*lua.LTable); ok {
			if s, ok := opts.RawGetString("shell").(lua.LString); ok && string(s) != "" {
				shellName = string(s)
			}
			if b, ok := opts.RawGetString("interactive").(lua.LBool); ok {
				interactive = bool(b)
			}
		}
	}
	spec, err := shell.Resolve(shellName, c.Registry)
	if err != nil {
		L.RaiseError("%v", err)
	}
	execCmd := exec.Command(spec.Path, spec.Argv(cmd)...)
	execCmd.Stdout = c.Stdout
	execCmd.Stderr = c.Stderr
	if interactive {
		execCmd.Stdin = c.Stdin
	}
	if err := execCmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			L.RaiseError("command failed (exit %d): %s", ee.ExitCode(), cmd)
		}
		L.RaiseError("command failed: %v: %s", err, cmd)
	}
	L.Push(lua.LTrue)
	return 1
}

func ctxExec(L *lua.LState) int {
	c := getCtx(L)
	name := L.CheckString(2)
	var args []string
	if L.GetTop() >= 3 {
		if t, ok := L.Get(3).(*lua.LTable); ok {
			t.ForEach(func(_, v lua.LValue) {
				args = append(args, lua.LVAsString(v))
			})
		}
	}
	execCmd := exec.Command(name, args...)
	execCmd.Stdout = c.Stdout
	execCmd.Stderr = c.Stderr
	if err := execCmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			L.RaiseError("exec failed (exit %d): %s %s", ee.ExitCode(), name, strings.Join(args, " "))
		}
		L.RaiseError("exec failed: %v: %s", err, name)
	}
	L.Push(lua.LTrue)
	return 1
}

func ctxRun(L *lua.LState) int {
	c := getCtx(L)
	name := L.CheckString(2)
	c.callDepth++
	if c.callDepth > maxCallDepth {
		L.RaiseError("ctx:run: recursion too deep (possible cycle calling %q)", name)
	}
	defer func() { c.callDepth-- }()
	var argsTable *lua.LTable
	if L.GetTop() >= 3 {
		if t, ok := L.Get(3).(*lua.LTable); ok {
			argsTable = t
		}
	}
	if argsTable == nil {
		argsTable = L.NewTable()
	}
	task, ok := c.File.Tasks[name]
	if !ok {
		L.RaiseError("ctx:run: task %q not found", name)
	}
	if err := callTask(L, c, task, argsTable); err != nil {
		L.RaiseError("%v", err)
	}
	L.Push(lua.LTrue)
	return 1
}

func callTask(L *lua.LState, c *Ctx, task *taskfile.Task, argsTable *lua.LTable) error {
	ctxUD := newCtxUserData(L, c)
	L.Push(task.Fn)
	L.Push(ctxUD)
	L.Push(argsTable)
	return L.PCall(2, 0, nil)
}

func ctxEcho(L *lua.LState) int {
	c := getCtx(L)
	n := L.GetTop()
	parts := make([]string, 0, n-1)
	for i := 2; i <= n; i++ {
		parts = append(parts, L.Get(i).String())
	}
	fmt.Fprintln(c.Stdout, strings.Join(parts, " "))
	return 0
}

func ctxAsk(L *lua.LState) int {
	c := getCtx(L)
	prompt := L.CheckString(2)
	def := ""
	if L.GetTop() >= 3 {
		def = lua.LVAsString(L.Get(3))
	}
	fmt.Fprintf(c.Stdout, "%s ", prompt)
	if c.Background || c.Stdin == nil {
		fmt.Fprintf(c.Stdout, "[no stdin: using default %q]\n", def)
		L.Push(lua.LString(def))
		return 1
	}
	if c.reader == nil {
		c.reader = bufio.NewReader(c.Stdin)
	}
	line, err := c.reader.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if err != nil && line == "" {
		L.Push(lua.LString(def))
		return 1
	}
	if line == "" {
		line = def
	}
	L.Push(lua.LString(line))
	return 1
}

func ctxCwd(L *lua.LState) int {
	_ = getCtx(L)
	wd, err := os.Getwd()
	if err != nil {
		L.RaiseError("cwd: %v", err)
	}
	L.Push(lua.LString(wd))
	return 1
}

func ctxOs(L *lua.LState) int {
	_ = getCtx(L)
	L.Push(lua.LString(runtime.GOOS))
	return 1
}
