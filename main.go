package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Println("tkx " + version)
		return
	}
	fmt.Print(usage)
}

const usage = `tkx - task runner (global tasks + background execution)

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

Tasks are defined in Taskfile.lua under the tkx config directory.
`
