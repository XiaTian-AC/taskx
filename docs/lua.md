# tkx Taskfile API 参考

tkx 的 `Taskfile.lua` 是 Lua 5.1 脚本（gopher-lua 解释）。本文档只列**tkx 暴露给 taskfile 作者的表面**——结构、`ctx` 方法、`args` 形状。假设你会写 Lua。

## 1. Taskfile 结构

```lua
local tasks = {
  -- task definitions here
}
return tasks
```

- 顶层必须是 `local tasks = { ... }` + `return tasks`
- 每个 task 名匹配 `[A-Za-z0-9][A-Za-z0-9_-]*`
- tkx 调用 `tkx <name>` 触发对应任务

## 2. Task 形状

两种形式：

### 简单形式
```lua
tasks.simple = function(ctx)
  ctx:echo("hi")
end
```
只用 `ctx`，无参数。

### 表格形式（带描述 + 参数声明）
```lua
tasks.release = {
  desc = "test, commit, tag, push",
  args = {
    tag   = { type = "string", required = false, desc = "git tag, e.g. v0.1.2" },
    force = { type = "bool",   required = false },
  },
  run = function(ctx, args)
    -- args.tag is string or nil
    -- args.force is true/false
  end,
}
```

`args` 声明字段（不写 = 任意 flag + 位置参数）：
| 字段 | 类型 | 说明 |
|---|---|---|
| `type` | `"string"` \| `"bool"`` | 必填，决定 `--x value` 还是 `--x` |
| `required` | bool | 缺失时 tkx 报错退出 1 |
| `desc` | string | `tkx help <task>` 显示 |

`type` 不写则默认为 `"string"`（可省去 `type` 字段）。

未在 `args` 声明的 flag：tkx 报错（strict 模式）。

## 3. `ctx` 方法

所有方法用冒号调用（`ctx:method(args)`）。方法签名：

| 方法 | 行为 | 错误时 |
|---|---|---|
| `ctx:sh(cmd, opts?)` | 通过 shell 执行 `cmd` | 非零 exit code 抛 Lua 错误 |
| `ctx:exec(name, args?)` | 直接执行 `name`，参数数组传给 exec（不经 shell） | 非零 exit code 抛错 |
| `ctx:run(name, args?)` | 调用同文件另一个 task，`args` 是 Lua table | task 不存在或循环调用 (>100 层) 抛错 |
| `ctx:echo(...)` | 打印到 stdout（多参数空格连接，自动换行） | — |
| `ctx:ask(prompt, default?)` | 读一行用户输入；后台任务无 stdin 时返回 default | — |
| `ctx:env(name)` | 读环境变量；不存在返回 nil | — |
| `ctx:cwd()` | 当前工作目录（绝对路径） | — |
| `ctx:os()` | OS 名：`"windows"` / `"linux"` / `"darwin"` | — |

`ctx:sh` 的 opts：
```lua
{ shell = "git", interactive = true }
```
- `shell`：覆盖本次 shell（从 `config.lua` 的 `shells` 表选）
- `interactive=true`：把 stdin 传给子命令（默认无）

`ctx:ask` 在 detached（bstart 启动的）任务里：`Background` 为 true 或 `Stdin == nil` → 自动返回 default。

## 4. `args` 形状

CLI 参数解析后变成 Lua table 传给任务函数（第二个参数）。

```lua
-- CLI: tkx release --tag v1.0 --force pos1 pos2
run = function(ctx, args)
  args.tag      -- "v1.0"   (string)
  args.force    -- true     (bool)
  args._        -- {"pos1", "pos2"}  (positional, array; or nil if none)
end
```

每条 flag 的值类型：
- 声明 `type="string"`：`--x VALUE` 或 `--x=VALUE` → string；`--x` 单独 → **报错**（需要值）
- 声明 `type="bool"`：`--x` → true；`--x=true/false` → bool；`--x value` → **报错**（不接受值）
- 未声明的 flag：报错

`--` 之后所有 token 进 `args._`（位置参数数组），即使看起来像 flag。

## 5. Lua 语言子集

tkx 用 gopher-lua（Lua 5.1）。taskfile 能用：

| 能用 | 备注 |
|---|---|
| `string`, `table`, `math`, `io`, `os`, `debug`, `utf8`, `coroutine` | gopher-lua 默认加载的标准库 |
| `pcall` / `xpcall` | 错误保护 |
| `string.format`, `string.gmatch` | 格式化 + 正则匹配 |
| `table.insert` / `table.remove` / `table.sort` | |
| `io.open` / `io.popen` / `io.read` | 文件 I/O |
| `os.execute` | 直接执行 shell 命令（不经 ctx，**自己负责错误处理**） |
| `goto` + label | 模拟 `continue` |

**不能用**（Lua 5.2+ 特性或 gopher-lua 未实现）：
- `string.pack` / `string.unpack`
- `goto` 以外的 `continue`
- 整数除法 `//`
- 整数类型（5.1 没区分 int/double）
- `require`（没有 `package.path`）
- `os.getenv`（用 `ctx:env` 替代）
- `os.exit`（用 `ctx:sh` 或 `error` 替代）

## 6. ctx 内部（不推荐直接用）

这些字段存在于 `ctx` 上但**不应在 taskfile 中直接读取**——只在 ctx 方法内部用：

| 字段 | 说明 |
|---|---|
| `ctx.File` | Go 侧 `*taskfile.File` 指针 |
| `ctx.L` | gopher-lua state |
| `ctx.ShellName` | 当前 shell 名 |
| `ctx.Registry` | shell 注册表 map |
| `ctx.Stdout` / `ctx.Stderr` / `ctx.Stdin` | I/O 流 |
| `ctx.Background` | 是否在 bstart 启动的后台任务里 |
| `ctx.callDepth` | ctx:run 递归深度（100 上限）|
| `ctx.reader` | bufio reader for ctx:ask |

如有合理需求需要其中之一，提 issue。

## 7. tkx 注入的全局

`tkx` 命名空间目前**没有**注入任何 Lua 全局符号。taskfile 里能看到的都是 Lua 标准库的（`string`, `table`, `math`, `io`, `os` 等）和上面列出的方法。如果未来需要加（如 `tkx.git.tag()`），会在这里更新。

## 8. 调试

```lua
-- 打印到 stderr（不影响 ctx:echo 的输出流）
io.stderr:write("debug: " .. tostring(x) .. "\n")

-- 看变量类型
ctx:echo(type(x))           -- "nil"/"string"/"table"/...
ctx:echo(#args._ or 0)      -- 数组长度（nil-safe）
```

错误信息带文件名 + 行号：
```
tkx: Taskfile.lua:7: attempt to index a nil value
```

`error("msg")` 在前台任务里抛错 → 退出码 1；在 detached（bstart）任务里抛错 → 退出码 1 + run.json 记 `exited(1)`。

## 9. 完整示例

```lua
local tasks = {
  -- 简单：只跑命令
  clean = function(ctx) ctx:sh("rm -rf dist/") end,

  -- 带参数：必填 + bool + 默认值
  release = {
    desc = "build and release",
    args = {
      tag   = { type = "string", required = true, desc = "git tag" },
      force = { type = "bool", required = false, desc = "force push" },
    },
    run = function(ctx, args)
      ctx:run("test")
      ctx:sh("git tag " .. args.tag)
      if args.force then
        ctx:sh("git push --force --tags")
      else
        ctx:sh("git push --tags")
      end
    end,
  },

  -- 平台分支 + 安全执行
  list = {
    desc = "list running processes (with the given name)",
    args = {
      name = { type = "string", required = true },
    },
    run = function(ctx, args)
      -- 用 ctx:exec 避免 shell 注入
      if ctx:os() == "windows" then
        ctx:exec("tasklist", {"/FI", "IMAGENAME eq " .. args.name .. ".exe"})
      else
        ctx:exec("pgrep", {"-l", args.name})
      end
    end,
  },
}
return tasks
```