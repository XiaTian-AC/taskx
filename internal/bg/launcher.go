package bg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type StartOptions struct {
	SelfPath  string
	TaskName  string
	TaskArgs  []string
	ShellName string
	CWD       string
	DataDir   string
}

func Start(opts StartOptions) (*Instance, error) {
	id, n, err := NextID(opts.DataDir, opts.TaskName)
	if err != nil {
		return nil, fmt.Errorf("assign instance ID: %w", err)
	}
	logPath := filepath.Join(opts.DataDir, "logs", fmt.Sprintf("%s#%d.log", opts.TaskName, n))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()
	argv := buildRunArgv(opts.TaskName, opts.ShellName, id, opts.TaskArgs)
	cmd := exec.Command(opts.SelfPath, argv...)
	cmd.Dir = opts.CWD
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("spawn failed: %w", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	inst := Instance{
		ID: id, Name: opts.TaskName, N: n, PID: pid, Args: opts.TaskArgs,
		CWD: opts.CWD, Started: time.Now().UTC(), Log: logPath, Status: "running",
	}
	if err := Register(opts.DataDir, inst); err != nil {
		_ = Stop(pid)
		return nil, fmt.Errorf("register instance: %w", err)
	}
	return &inst, nil
}

func buildRunArgv(taskName, shellName, id string, taskArgs []string) []string {
	argv := []string{"_run", taskName}
	if shellName != "" {
		argv = append(argv, "--shell", shellName)
	}
	argv = append(argv, "--id", id, "--")
	argv = append(argv, taskArgs...)
	return argv
}
