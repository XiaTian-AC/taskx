# tkx

A modern task runner with Lua taskfiles and built-in background task management.

## Why tkx?

- **Tasks are real code** - `Taskfile.lua` tasks are Lua functions with native conditionals, parameters, and inter-task calls. No YAML.
- **Global tasks** - define tasks once in `~/.config/taskx/Taskfile.lua`, run them from any directory.
- **Background execution** - `tkx bstart` runs tasks detached from the terminal; `tkx watch` tails the output live; `tkx stop` kills them.
- **Single binary** - zero runtime dependencies. Just download and run.

## Quick Start

1. Build:
   ```
   go build -o tkx.exe .
   ```

2. Create `~/.config/taskx/Taskfile.lua`:
   ```lua
   local tasks = {
     hello = function(ctx)
       ctx:echo("hello world")
     end,

     release = {
       desc = "test, commit, tag, push",
       args = {
         tag = { type = "string", required = false, desc = "git tag, e.g. v0.1.2" },
       },
       run = function(ctx, args)
         ctx:sh("go test ./...")
         ctx:sh("git add . && git commit -m 'release'")
         if args.tag then
           ctx:sh("git tag " .. args.tag)
         end
         ctx:sh("git push --tags")
       end,
     },
   }
   return tasks
   ```

3. Run:
   ```
   tkx hello
   tkx release --tag v0.1.0
   tkx bstart release --tag v0.2.0
   tkx ls-running
   tkx watch release#1
   tkx stop release#1
   ```

## Commands

| Command | Description |
|---|---|
| `tkx ls` | List tasks |
| `tkx <task> [args]` | Run a task (foreground) |
| `tkx bstart <task> [args]` | Run a task (background, detached) |
| `tkx ls-running` | List background instances |
| `tkx watch <name>[#N]` | Tail a background instance's log live |
| `tkx stop <name>[#N]` | Stop background instance(s) |
| `tkx help [task]` | Help or task details |

## Shell Configuration

tkx uses `pwsh` on Windows and `bash` on Unix by default. Register custom shells in `~/.config/taskx/config.lua`:

```lua
return {
  shells = {
    bash = "C:\\Program Files\\Git\\bin\\bash.exe",
    pwsh = "C:\\Program Files\\PowerShell\\7\\pwsh.exe",
  },
}
```

Select a shell per-run with `--shell`:
```
tkx build --shell bash
```

## Roadmap

- Project-level Taskfile (upward search + project overrides global)
- just / make one-click migration
- Parallel task execution (`ctx:run_parallel`)
- Shell completion
