# tkx

A modern task runner with Lua taskfiles and built-in background task management.

## Installation

**Scoop** (Windows):
```
scoop bucket add XiaTian-AC https://github.com/XiaTian-AC/XiaTian-AC-bucket
scoop install tkx
```

**Homebrew** (macOS/Linux):
```
brew install XiaTian-AC/tap/tkx
```

**Direct download:** grab a binary from [Releases](https://github.com/XiaTian-AC/tkx/releases).

## Why tkx?

- **Tasks are real code** - `Taskfile.lua` tasks are Lua functions with native conditionals, parameters, and inter-task calls. No YAML.
- **Global tasks** - define tasks once in `~/.config/tkx/Taskfile.lua`, run them from any directory.
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

## Security Notes

- `ctx:sh(cmd)` runs `cmd` through a shell (pwsh/bash). The command string is passed as-is — **if you interpolate user-controlled values, you are responsible for escaping/sanitizing them** to prevent command injection.
- For safe execution of a known command with arguments, use `ctx:exec(name, args_table)` which bypasses the shell entirely (no string interpolation, no injection).
- `Taskfile.lua` is executed as a Lua script with full capabilities (file IO, `os.execute`, etc.). Only run taskfiles from sources you trust.

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

## Argument Parsing Notes

- `--flag value` — `value` is consumed as the flag's value (must not start with `-`).
- `--flag=value` — inline form, always works.
- `--flag` alone — bool flag (true).
- Negative numbers: `--offset -5` treats `-5` as a separate token (positional or unknown flag). Use `--offset=-5` instead.
- `--` — everything after is positional, even if it looks like flags.

## Display & Cleanup

`tkx ls-running` filters and sorts instances using rules in `config.lua`:

```lua
return {
  shells = { ... },
  display = {
    ls_running = {
      time = "1h",          -- "0" = running only; "30m"/"1h"/"2d"/"1w" = ended window
      running_first = true, -- running on top of ended
      newest_first = true,  -- newest first within each group
    },
  },
}
```

`time` accepts `0` (running only), `Nm`/`Nh`/`Nd`/`Nw`, or `N` (default hours).

Clean up old entries:

```
tkx clean              # remove all ended instances and their logs
tkx clean --older-than=7d   # remove only entries ended > 7 days ago
tkx clean --all            # alias for --older-than=0 (same as no flag)
```

## Roadmap

- Project-level Taskfile (upward search + project overrides global)
- just / make one-click migration
- Parallel task execution (`ctx:run_parallel`)
- Shell completion
