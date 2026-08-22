# tkx — Design Spec v1

**Date:** 2026-08-21
**Status:** Approved → implementation plan written
**Stack:** Go + gopher-lua (embedded Lua 5.1), single self-contained binary, zero runtime deps

## 1. Overview

`tkx` is a modern task runner benchmarked against go-task / just / make, with two differentiators:

1. **Task files are real code** — `Taskfile.lua` tasks are Lua functions with native conditionals, parameters, and inter-task calls (not YAML).
2. **Built-in background task management** — `tkx bstart` runs a task detached from the terminal; `tkx ls-running` / `tkx watch` / `tkx stop` manage it.

**v1 scope:** global Taskfile only (runs from any directory) + parameter passing + full background task suite. Project-level Taskfiles and just/make migration are roadmap.

## 2. Confirmed Decisions

| Decision | Choice |
|---|---|
| CLI implementation | Go + gopher-lua, single binary, zero deps |
| Task language | Lua (embedded, startup <5ms) |
| Taskfile shape | `tasks` table + `ctx` object (form 1) |
| Platform | Cross-platform (pwsh + bash) |
| Command name | `tkx` |
| Config dir | `$XDG_CONFIG_HOME/taskx` (default `~/.config/taskx`) |
| Data dir | `$XDG_DATA_HOME/taskx` (default `~/.local/share/taskx`) |
| Task discovery (v1) | Global only; project-level in v2 |
| Parameter model | Generic `--flag` parsing into `args` table + optional per-task declaration |
| Background instances | Multi-instance allowed; same-name instances trigger risk warning + y/N confirm |
| Shell model | Named shell registry in global config; default Win→pwsh / Unix→bash |

## 3. Architecture

Single re-entrant Go binary, no daemon.

```
~/.config/taskx/
  Taskfile.lua    # global task definitions
  config.lua      # global config: shells registry

~/.local/share/taskx/
  run.json        # background instance registry
  logs/
    <name>#<N>.log  # per-instance output log
```

**Background = self-fork:** `tkx bstart build` spawns a detached child `tkx _run build` (survives terminal close), child loads Taskfile, runs task, streams stdout/stderr to log file, updates run.json on exit. Windows: `CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS`. Unix: `setsid`.

## 4. Command Surface

| Command | Description |
|---|---|
| `tkx ls` | List global tasks (name + desc) |
| `tkx <name> [args]` | Run task foreground (default) |
| `tkx start <name> [args]` | Explicit foreground run |
| `tkx bstart <name> [args]` | Run detached in background |
| `tkx ls-running` | List background instances (filtered by `display.ls_running.time` window) |
| `tkx clean [--all\|--older-than <dur>]` | Remove ended instances and their logs |
| `tkx watch <name>[#N]` | Tail instance log live; Ctrl+C exits viewer only |
| `tkx stop <name>[#N]` | Stop one instance; `stop <name>` stops all (confirm if >1) |
| `tkx run <name> [args]` | Force-run task (builtin-name collision escape hatch) |
| `tkx help [task]` | Help or task details |
| `tkx version` | Print version |
| `tkx _run <name> ...` | Internal command for bstart fork (not advertised) |

Builtin names reserved; `tkx run <name>` forces task execution on collision.

## 5. Taskfile.lua Authoring Model

```lua
local tasks = {
  build = function(ctx)
    ctx:sh("cargo build --release")
  end,

  release = {
    desc = "test, commit, tag, push",
    args = {
      tag   = { type = "string", required = false, desc = "git tag, e.g. v0.1.2" },
      force = { type = "bool",   required = false, desc = "force push" },
    },
    run = function(ctx, args)
      ctx:run("test")
      ctx:sh("git add . && git commit -m 'release'")
      if args.tag then ctx:sh("git tag " .. args.tag) end
      if args.force then ctx:sh("git push --force") else ctx:sh("git push") end
      ctx:sh("git push --tags")
    end,
  },
}
return tasks
```

**ctx API (v1):**
- `ctx:sh(cmd, opts?)` — run shell; non-zero exit raises Lua error; `opts = {shell="name", interactive=true}`
- `ctx:run(name, args?)` — call another task in same file
- `ctx:echo(...)` / `ctx:ask(prompt, default?)` / `ctx:env(name)` / `ctx:cwd()` / `ctx:os()`

## 6. Parameter Model

- **Generic parse:** `--tag v0.1.2` → `args.tag="v0.1.2"`; `--force` → `args.force=true`; positional → `args._` array; unknown flag → error (when task declares args spec)
- **Optional declaration** (table-form `args` field): `type` (string/bool) / `required` / `desc` → validation + `tkx help <task>` display
- **Passthrough:** `tkx <name> --shell gitbash` — CLI-level flags consumed by tkx first, rest go to `args`

## 7. Background Mechanism (Multi-instance + Safety Gate)

- **Instance IDs:** `name#N` (N = per-name incrementing counter, persists across restarts); log at `logs/<name>#<N>.log`
- **bstart safety gate:** same-name running instances exist → print count + instance list + risk warning (file/port contention, unlocked git ops) → y/N confirm → only then launch
- **watch:** `watch <name>#N` exact; `watch <name>` = newest instance; prints full log then follows appends; Ctrl+C exits viewer only; exited instance → drain + print exit code
- **stop:** exact one, or all by name (>1 → confirm); kills process tree; marks status stopped
- **Lifecycle:** exited instances retained for ls-running/watch history; logs rotated (deleted) when same name is bstarted again

## 8. Shell Model

- **Default:** Windows → `pwsh -NoProfile -Command <cmd>`; Unix → `bash -euo pipefail -c <cmd>`
- **Named registry** (`config.lua`, any OS):
  ```lua
  return {
    shells = {
      bash   = "C:\\Program Files\\Git\\bin\\bash.exe",
      pwsh   = "C:\\Program Files\\PowerShell\\7\\pwsh.exe",
      mysh   = "/opt/mytool/bin/mysh",
    },
  }
  ```
- **Builtin names:** `pwsh` / `bash` / `sh`. Windows `bash` detection chain: **① `where.exe git` → `<git install root>/bin/bash.exe` → ② common paths (Program Files, LOCALAPPDATA, scoop) → ③ error prompting config.lua registration**. Unix: `bash`/`pwsh` via PATH.
- **Invocation args by name:** pwsh → `[-NoProfile, -Command]`; bash → `[-euo, pipefail, -c]`; sh → `[-e, -c]`; registered custom → `[-c]`
- **Selection priority:** `ctx:sh(opts.shell)` > CLI `--shell` > OS default

## 8b. Display & Cleanup (v1.2)

`tkx ls-running` 默认过滤+排序，规则在 `config.lua`：
- `display.ls_running.time` — 结束实例的时间窗口（默认 `1h`，`0` = 只显示 running）
- `display.ls_running.running_first` — running 在前（默认 true）
- `display.ls_running.newest_first` — 同组内最新在前（默认 true）

时间格式：`0` / `30m` / `1h` / `2d` / `1w`，纯数字默认小时。

`tkx clean` 清理已结束的实例和日志：
- `tkx clean` — 全部已结束实例
- `tkx clean --older-than=7d` — 仅清理结束 > 7 天的
- `tkx clean --all` — 等价于 `--older-than=0`

## 9. Error Handling

- Lua error → formatted message (with task name) + exit code 1
- `ctx:sh` failure → Lua error with command + exit code
- Background exit code → written to run.json; visible via `watch` and `ls-running`
- Usage errors → exit code 2

## 10. Testing

- **Unit:** argparse (flag syntax, types, unknown-flag, coercion), registry (NextID, rotation, concurrent locking), taskfile loader (valid/invalid Lua, function/table forms, name validation), shell resolution (argsFor, registry override, Windows detection chain with injection)
- **Integration:** sample Taskfile.lua through foreground run / bstart → ls-running → watch → stop full flow; verify log content + exit codes; multi-instance confirm gate; log rotation
- **Cross-platform:** taskkill vs kill, process group, detach flags abstracted via build tags

## 11. Roadmap (v2+)

1. Project-level Taskfile: upward search + project overrides global
2. just / make one-click migration: parse justfile / Makefile → generate Taskfile.lua
3. ctx enhancements: `ctx:run_parallel`, colored output / progress, shell completion
