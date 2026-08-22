package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"tkx/internal/argparse"
	"tkx/internal/bg"
	"tkx/internal/config"
	"tkx/internal/runtime"
	"tkx/internal/taskfile"
)

type Deps struct {
	SelfPath string
	Version  string
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
}

func Run(argv []string, d Deps) int {
	if d.Version == "" {
		d.Version = "dev"
	}
	if d.SelfPath == "" {
		d.SelfPath, _ = os.Executable()
	}
	if d.Stdin == nil {
		d.Stdin = os.Stdin
	}
	if d.Stdout == nil {
		d.Stdout = os.Stdout
	}
	if d.Stderr == nil {
		d.Stderr = os.Stderr
	}

	if len(argv) == 0 {
		usage(d.Stdout)
		return 2
	}

	cmd, rest := argv[0], argv[1:]
	switch cmd {
	case "help", "--help", "-h":
		if len(rest) > 0 {
			return cmdHelpTask(rest[0], d)
		}
		usage(d.Stdout)
		return 0
	case "version", "--version":
		fmt.Fprintf(d.Stdout, "tkx %s\n", d.Version)
		return 0
	case "ls":
		return cmdLs(d)
	case "start", "run":
		if len(rest) == 0 {
			fmt.Fprintf(d.Stderr, "tkx: usage: tkx %s <task> [args]\n", cmd)
			return 2
		}
		return cmdRunTask(rest, d)
	case "bstart":
		if len(rest) == 0 {
			fmt.Fprintln(d.Stderr, "tkx: usage: tkx bstart <task> [args]")
			return 2
		}
		return cmdBstart(rest, d)
	case "ls-running":
		return cmdLsRunning(d)
	case "watch":
		if len(rest) == 0 {
			fmt.Fprintln(d.Stderr, "tkx: usage: tkx watch <name>[#N]")
			return 2
		}
		return cmdWatch(rest[0], d)
	case "stop":
		if len(rest) == 0 {
			fmt.Fprintln(d.Stderr, "tkx: usage: tkx stop <name>[#N]")
			return 2
		}
		return cmdStop(rest[0], d)
	case "_run":
		if len(rest) == 0 {
			return 2
		}
		return cmdInternalRun(rest, d)
	}
	return cmdRunTask(argv, d)
}

func usage(out io.Writer) {
	fmt.Fprint(out, `tkx - task runner (global tasks + background execution)

Usage:
  tkx <task> [args]        run a task in the foreground
  tkx bstart <task> [args] run a task in the background (detached)
  tkx ls                   list tasks
  tkx ls-running           list background instances
  tkx watch <name>[#N]     tail a background instance's log live
  tkx stop <name>[#N]      stop background instance(s)
  tkx run <task> [args]    force-run a task (builtin-name escape hatch)
  tkx help [task]          help (or task details)
  tkx version              print version

Flags:
  --shell <name>           select shell for this run

Tasks are defined in Taskfile.lua under the tkx config directory.
`)
}

func loadTaskfile(d Deps) (*taskfile.File, map[string]string, error) {
	cfgDir, err := config.ConfigDir()
	if err != nil {
		return nil, nil, err
	}
	shells, err := config.Shells(cfgDir)
	if err != nil {
		return nil, nil, err
	}
	tfPath, err := config.TaskfilePath()
	if err != nil {
		return nil, nil, err
	}
	f, err := taskfile.Load(tfPath)
	if err != nil {
		return nil, nil, err
	}
	return f, shells, nil
}

func extractShell(argv []string) (string, []string) {
	for i := 0; i < len(argv); i++ {
		if argv[i] == "--" {
			break
		}
		if argv[i] == "--shell" && i+1 < len(argv) {
			rest := append(append([]string{}, argv[:i]...), argv[i+2:]...)
			return argv[i+1], rest
		}
		if strings.HasPrefix(argv[i], "--shell=") {
			rest := append(append([]string{}, argv[:i]...), argv[i+1:]...)
			return strings.TrimPrefix(argv[i], "--shell="), rest
		}
	}
	return "", argv
}

func cmdLs(d Deps) int {
	f, _, err := loadTaskfile(d)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	defer f.Close()
	if len(f.Order) == 0 {
		fmt.Fprintln(d.Stdout, "no tasks defined")
		return 0
	}
	maxLen := 0
	for _, name := range f.Order {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}
	for _, name := range f.Order {
		task := f.Tasks[name]
		fmt.Fprintf(d.Stdout, "%-*s  %s\n", maxLen, name, task.Desc)
	}
	return 0
}

func cmdRunTask(argv []string, d Deps) int {
	name := argv[0]
	shellName, taskArgs := extractShell(argv[1:])
	f, shells, err := loadTaskfile(d)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	defer f.Close()
	task, ok := f.Get(name)
	if !ok {
		fmt.Fprintf(d.Stderr, "tkx: task %q not found; run 'tkx ls'\n", name)
		return 1
	}
	parsed, err := argparse.Parse(taskArgs, task.Args)
	if err != nil {
		fmt.Fprintf(d.Stderr, "tkx: task %q: %v\n", name, err)
		return 1
	}
	argsTable := argparse.ToLuaTable(f.L, parsed)
	err = runtime.Run(f, name, argsTable, runtime.Options{
		ShellName: shellName, Registry: shells,
		Stdout: d.Stdout, Stderr: d.Stderr, Stdin: d.Stdin,
	})
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	return 0
}

func cmdBstart(argv []string, d Deps) int {
	name := argv[0]
	shellName, taskArgs := extractShell(argv[1:])
	f, shells, err := loadTaskfile(d)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	defer f.Close()
	task, ok := f.Get(name)
	if !ok {
		fmt.Fprintf(d.Stderr, "tkx: task %q not found; run 'tkx ls'\n", name)
		return 1
	}
	_, err = argparse.Parse(taskArgs, task.Args)
	if err != nil {
		fmt.Fprintf(d.Stderr, "tkx: task %q: %v\n", name, err)
		return 1
	}
	dataDir, err := config.EnsureDataDir()
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	running, err := bg.Running(dataDir, name)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	if len(running) > 0 {
		fmt.Fprintf(d.Stdout, "%d instance(s) of %q already running:\n", len(running), name)
		for _, inst := range running {
			fmt.Fprintf(d.Stdout, "  %s (pid %d, started %s)\n", inst.ID, inst.PID, inst.Started.Format("15:04:05"))
		}
		fmt.Fprintln(d.Stdout, "WARNING: same-name instances may conflict (file/port contention, unlocked git operations).")
		if !confirm("Start another?", d.Stdin, d.Stdout) {
			fmt.Fprintln(d.Stdout, "aborted")
			return 0
		}
	}
	_ = shells
	cwd, _ := os.Getwd()
	inst, err := bg.Start(bg.StartOptions{
		SelfPath:  d.SelfPath,
		TaskName:  name,
		TaskArgs:  taskArgs,
		ShellName: shellName,
		CWD:       cwd,
		DataDir:   dataDir,
	})
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	fmt.Fprintf(d.Stdout, "started %s (pid %d)\nlog: %s\nwatch: tkx watch %s\nstop:  tkx stop %s\n",
		inst.ID, inst.PID, inst.Log, inst.ID, inst.ID)
	return 0
}

func cmdLsRunning(d Deps) int {
	dataDir, err := config.DataDir()
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	all, err := bg.Snapshot(dataDir)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	if len(all) == 0 {
		fmt.Fprintln(d.Stdout, "no background instances")
		return 0
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Started.After(all[j].Started)
	})
	for _, inst := range all {
		status := inst.Status
		if status == "running" && !bg.Alive(inst.PID) {
			status = "dead"
			_ = bg.MarkEnded(dataDir, inst.ID, "dead", -1)
		}
		if inst.Status == "exited" && inst.ExitCode != 0 {
			status = fmt.Sprintf("exited(%d)", inst.ExitCode)
		}
		age := time.Since(inst.Started).Round(time.Second)
		fmt.Fprintf(d.Stdout, "%-16s  pid %-7d  %-12s  %s  %s\n",
			inst.ID, inst.PID, status, age, inst.CWD)
	}
	return 0
}

func cmdWatch(ref string, d Deps) int {
	dataDir, err := config.DataDir()
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	all, err := bg.Snapshot(dataDir)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	inst, err := findInstance(all, ref)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	isAlive := func() bool {
		return inst.Status == "running" && bg.Alive(inst.PID)
	}
	if err := bg.Watch(inst.Log, inst.ID, isAlive, d.Stdout); err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	return 0
}

func cmdStop(ref string, d Deps) int {
	dataDir, err := config.DataDir()
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	all, err := bg.Snapshot(dataDir)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	var targets []bg.Instance
	if strings.Contains(ref, "#") {
		for _, inst := range all {
			if inst.ID == ref && inst.Status == "running" {
				targets = append(targets, inst)
				break
			}
		}
		if len(targets) == 0 {
			for _, inst := range all {
				if inst.ID == ref {
					fmt.Fprintf(d.Stdout, "%s: already %s\n", inst.ID, inst.Status)
					return 0
				}
			}
			fmt.Fprintf(d.Stderr, "tkx: no instance %q\n", ref)
			return 1
		}
	} else {
		for _, inst := range all {
			if inst.Name == ref && inst.Status == "running" {
				targets = append(targets, inst)
			}
		}
		if len(targets) == 0 {
			fmt.Fprintf(d.Stdout, "no running instances of %q\n", ref)
			return 0
		}
		if len(targets) > 1 {
			fmt.Fprintf(d.Stdout, "stopping %d instances of %q:\n", len(targets), ref)
			for _, inst := range targets {
				fmt.Fprintf(d.Stdout, "  %s (pid %d)\n", inst.ID, inst.PID)
			}
			if !confirm("Stop all?", d.Stdin, d.Stdout) {
				fmt.Fprintln(d.Stdout, "aborted")
				return 0
			}
		}
	}
	for _, inst := range targets {
		if !bg.Alive(inst.PID) {
			fmt.Fprintf(d.Stdout, "%s: process already gone\n", inst.ID)
		} else if err := bg.Stop(inst.PID); err != nil {
			fmt.Fprintf(d.Stderr, "tkx: stop %s: %v\n", inst.ID, err)
		} else {
			fmt.Fprintf(d.Stdout, "stopped %s\n", inst.ID)
		}
		_ = bg.MarkEnded(dataDir, inst.ID, "stopped", -1)
	}
	return 0
}

func cmdInternalRun(argv []string, d Deps) int {
	name := argv[0]
	rest := argv[1:]
	shellName := os.Getenv("TKX_SHELL")
	id := os.Getenv("TKX_INSTANCE_ID")
	cwd := os.Getenv("TKX_CWD")
	var taskArgs []string
	for i := 0; i < len(rest); {
		switch {
		case rest[i] == "--shell" && i+1 < len(rest):
			shellName = rest[i+1]
			i += 2
		case rest[i] == "--id" && i+1 < len(rest):
			id = rest[i+1]
			i += 2
		case rest[i] == "--":
			taskArgs = append(taskArgs, rest[i+1:]...)
			i = len(rest)
		default:
			taskArgs = append(taskArgs, rest[i])
			i++
		}
	}
	if cwd != "" {
		os.Chdir(cwd)
	}
	f, shells, err := loadTaskfile(d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tkx: %v\n", err)
		return 1
	}
	defer f.Close()
	task, ok := f.Get(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "tkx: task %q not found\n", name)
		return 1
	}
	parsed, err := argparse.Parse(taskArgs, task.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tkx: %v\n", err)
		return 1
	}
	argsTable := argparse.ToLuaTable(f.L, parsed)
	exitCode := 0
	err = runtime.Run(f, name, argsTable, runtime.Options{
		ShellName: shellName, Registry: shells,
		Stdout: os.Stdout, Stderr: os.Stderr, Stdin: nil,
		Background: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "tkx: %v\n", err)
		exitCode = 1
	}
	if id != "" {
		dataDir, _ := config.DataDir()
		_ = bg.MarkEnded(dataDir, id, "exited", exitCode)
	}
	return exitCode
}

func cmdHelpTask(name string, d Deps) int {
	f, _, err := loadTaskfile(d)
	if err != nil {
		fmt.Fprintln(d.Stderr, "tkx:", err)
		return 1
	}
	defer f.Close()
	task, ok := f.Get(name)
	if !ok {
		fmt.Fprintf(d.Stderr, "tkx: task %q not found\n", name)
		return 1
	}
	fmt.Fprintf(d.Stdout, "tkx %s\n", name)
	if task.Desc != "" {
		fmt.Fprintf(d.Stdout, "  %s\n", task.Desc)
	}
	if task.Args != nil {
		fmt.Fprintln(d.Stdout, "  arguments:")
		argNames := make([]string, 0, len(task.Args))
		for k := range task.Args {
			argNames = append(argNames, k)
		}
		sort.Strings(argNames)
		for _, argName := range argNames {
			sp := task.Args[argName]
			req := ""
			if sp.Required {
				req = " (required)"
			}
			desc := sp.Desc
			if desc == "" {
				desc = sp.Type
			}
			fmt.Fprintf(d.Stdout, "    --%s %s%s  %s\n", argName, sp.Type, req, desc)
		}
	}
	return 0
}

func confirm(prompt string, in io.Reader, out io.Writer) bool {
	fmt.Fprintf(out, "%s [y/N] ", prompt)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if err != nil && line == "" {
		return false
	}
	return line == "y" || line == "yes"
}

func findInstance(all []bg.Instance, ref string) (bg.Instance, error) {
	if strings.Contains(ref, "#") {
		for i := range all {
			if all[i].ID == ref {
				return all[i], nil
			}
		}
		return bg.Instance{}, fmt.Errorf("no instance %q", ref)
	}
	var newest *bg.Instance
	for i := range all {
		if all[i].Name == ref {
			if newest == nil || all[i].N > newest.N {
				newest = &all[i]
			}
		}
	}
	if newest == nil {
		return bg.Instance{}, fmt.Errorf("no instance for %q", ref)
	}
	return *newest, nil
}
