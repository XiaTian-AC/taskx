# tkx Taskfile.lua Lua 速查手册

tkx 的 `Taskfile.lua` 是 Lua 5.1 脚本（gopher-lua 解释执行）。本手册只讲你写 taskfile 时会用到的 Lua 子集 + tkx 特有 API，不需要先学完整 Lua。

## 1. 速查表

| 你想做的事 | Lua 写法 |
|---|---|
| 字符串拼接 | `"hi " .. name` |
| 字符串包含 | `string.find(s, "v1.0") ~= nil` 或 `s:find("v1.0")` |
| 字符串前缀 | `s:sub(1, 4) == "v1.0"` |
| 数字转字符串 | `tostring(42)` |
| 字符串转数字 | `tonumber("42")` |
| 取字符串长度 | `#s` 或 `string.len(s)` |
| 切片 | `s:sub(1, 5)` |
| 大写/小写 | `s:upper()` / `s:lower()` |
| 拆分 | `for w in string.gmatch(s, "%S+") do ... end` |
| 替换 | `s:gsub("foo", "bar")`（注意：gsub 返回 (new_str, count)）|
| 格式化 | `string.format("v%d.%d", 1, 2)` |
| 表长度 | `#t`（只对数组部分）|
| 表是否有 key | `t.key ~= nil` |
| 遍历表 | `for k, v in pairs(t) do ... end` |
| 遍历数组 | `for i, v in ipairs(arr) do ... end` |
| 数组追加 | `table.insert(arr, item)` |
| 数组弹出 | `table.remove(arr)`  |
| 类型 | | `type(x)` → `"nil"`/`number`/`string`/`boolean`/`table`/`function`/`userdata`/`thread` |

## 2. 最小 Taskfile

```lua
local tasks = {
  hello = function(ctx)
    ctx:echo("hello world")
  end,
}
return tasks
```

- 必须 `local tasks = { ... }` 定义一个 table
- 必须 `return tasks`（这是 taskfile loader 期望的返回值）
- 每个 task 是函数 `function(ctx)` 或带元信息的 table（见 §5）
- 函数名（`hello`）就是 CLI 调用的任务名，只能是 `[A-Za-z0-9_-]+`

## 3. 变量与类型

Lua 是动态类型。八种类型：`nil` / `boolean` / `number` / `string` / `function` / `table` / `userdata` / `thread`。taskfile 里基本只用前六种。

```lua
local n = 42           -- number（Lua 5.1 没区分 int/double，都是 double）
local s = "hello"      -- string，单引号双引号都行
local b = true         -- boolean
local t = {1, 2, 3}    -- table（既是数组又是 map）
local f = function() end  -- function
local x = nil          -- nil 表示"无值"
```

注意：
- Lua 中 **只有 `nil` 和 `false` 是假**，其他都是真（包括 0 和空字符串）
- `local` 声明局部变量；不写 `local` 则是全局变量（**容易踩坑**，一定要写 `local`）
- 未声明的全局变量值为 `nil`，读取不报错

## 4. 字符串

```lua
local s = "hello"
local upper = s:upper()              -- "HELLO"
local sub = s:sub(1, 3)              -- "hel"（1-indexed，含端点）
local contains = s:find("ll")        -- 3, 4（如果找到）或 nil
local replaced = s:gsub("l", "L")    -- "heLLo", 2（返回新串 + 替换次数）
local parts = {}; for w in s:gmatch("%w+") do table.insert(parts, w) end
```

**关键点**：Lua 字符串是 **byte sequence**，不是 Unicode code point。中文字符每个占 3 字节（UTF-8），`#s` 返回字节数，`string.sub` 按字节切。如果要按 rune 切片需要自实现或避免。

数字 → 字符串：
```lua
tostring(42)            -- "42"
string.format("%d", 42)  -- "42"
string.format("%.2f", 3.14159)  -- "3.14"
```

## 5. 表（Table）

Lua 唯一的数据结构。同时是数组（连续整数 key）和 map（任意 key）。

```lua
local t = {}                          -- 空表
t.name = "alice"                      -- 字符串 key → map 风格
t[1] = "first"                        -- 数字 key → 数组 风格
t["with space"] = 42                  -- 含特殊字符的 key 用 []
print(t.name, t[1], t["with space"])
```

数组写法：
```lua
local arr = {"a", "b", "c"}           -- 等价于 {arr[1]="a", arr[2]="b", arr[3]="c"}
arr[#arr + 1] = "d"                   -- 追加（Lua 5.1 没有 # 操作符 O(1) 警告，table 连续即可）
table.insert(arr, "e")                -- 同样的事，更清晰
table.insert(arr, 2, "B")             -- 在 index 2 处插入，B 和后面的元素后移
```

遍历：
```lua
for k, v in pairs(t) do
  print(k, v)            -- 无序，包含所有 key
end

for i, v in ipairs(arr) do
  print(i, v)            -- 1..N，按数组顺序，遇到 nil 停止
end
```

常用操作：
```lua
t.key = nil            -- 删除 key
t.key = nil            -- 第二次 nil 不会报错（幂等）
t.x, t.y = 1, 2         -- 多重赋值
a, b = b, a            -- swap
local a, b = f()        -- 多返回值（taskfile 里基本用不到）
```

## 6. 函数

```lua
local function greet(name)
  return "hi " .. name
end
local msg = greet("alice")

-- 匿名函数 / 闭包
local mk = function(prefix)
  return function(suffix)
    return prefix .. " " .. suffix
  end
end
local f = mk("v1")
print(f("ready"))   -- "v1 ready"
```

调用约定：
- 多参数：`print("a", "b", "c")`
- 单 string/table 方法：`s:upper()` 等价于 `string.upper(s)`
- 多返回值：`local a, b = string.find(s, "x")`（找不到时 a/b 是 nil）

**闭包**：Lua 函数是 first-class 值，可以捕获外层变量。常用于：
```lua
local counter = 0
local function tick()
  counter = counter + 1
  return counter
end
-- tick() 每次调用 counter 都加 1
```

## 7. 控制流

```lua
-- if
if x == 0 then
  ...
elseif x > 0 then
  ...
else
  ...
end

-- while
while n > 0 do
  n = n - 1
end

-- repeat-until（条件后置，至少执行一次）
repeat
  n = n - 1
until n == 0

-- numeric for（包含两端）
for i = 1, 10 do
  ...
end
-- step（默认 1，可负）
for i = 10, 1, -1 do
  ...
end

-- 通用 for
for k, v in pairs(t) do
  ...
end
```

注意：Lua 的 `for` 循环变量是 **循环体内的局部变量**，循环外不可见。

`break` 只能跳出一层循环，没有 `continue`（5.1）。如果需要"跳过本轮"用 `goto` + label：
```lua
for _, x in ipairs(arr) do
  if x.skip then goto continue end
  -- 处理 x
  ::continue::
end
```

## 8. 错误处理

taskfile 里错误信息会显示给用户，进程退出码 1。

```lua
-- 抛错（栈跟踪包含行号）
error("something went wrong")
error("bad input: " .. value)  -- 拼接后抛

-- 断言
assert(type(x) == "string", "expected string, got " .. type(x))

-- pcall：保护调用
local ok, err = pcall(function() risky() end)
if not ok then
  ctx:echo("failed: " .. err)
  -- 可以继续执行
end
```

**常见错误信息对照：**
| 错误 | 原因 |
|---|---|
| `attempt to index a nil value (global 'X')` | 用了未定义的全局变量 |
| `attempt to perform arithmetic on a nil value` | number 变量是 nil |
| `bad argument #1 to 'X' (string expected, got nil)` | 函数参数类型错 |
| `'=' expected near 'X'` | 语法错误（常见：中文标点 / `:` vs `..`）|
| `module 'X' not found` | `require` 的模块不存在（taskfile 里基本用不到 require）|

## 9. tkx 特有：任务函数

两种形式：

### 简单形式
```lua
tasks.simple = function(ctx)
  ctx:echo("hi")
end
```
只用 ctx，没参数。

### 表格形式（带元信息 + 参数声明）
```lua
tasks.release = {
  desc = "test, commit, tag, push",              -- 任务描述（tkx ls / tkx help 显示）
  args = {                                       -- 参数声明（可选）
    tag = { type = "string", required = false, desc = "git tag, e.g. v0.1.2" },
    force = { type = "bool", required = false },
  },
  run = function(ctx, args)
    -- args.tag 是 string 或 nil
    -- args.force 是 true/false
  end,
}
```

`args` 声明影响 CLI 行为：
- `--tag v1.0` → `args.tag = "v1.0"`
- `--force` → `args.force = true`（不传值时）
- `--force=false` → `args.force = false`
- 必填项缺失 → 报错退出
- 未声明的 flag → 报错
- `--` 之后的 token 全部进 `args._`（位置参数数组）

## 10. tkx ctx 方法

所有 ctx 方法都是 **冒号语法**（自动传 ctx 作 self）。错误情况（命令非零退出、参数校验失败）抛 Lua 错误，进程退出 1。

| 方法 | 用途 | 备注 |
|---|---|---|
| `ctx:sh(cmd, opts?)` | 通过 shell 执行命令 | 非零退出抛错。`opts = {shell="git", interactive=true}` 可覆盖 shell / 启用 stdin |
| `ctx:exec(name, args?)` | 直接执行二进制（不经 shell） | 安全，无字符串注入风险 |
| `ctx:run(name, args?)` | 调用同文件另一个 task | `args` 是 Lua table。循环调用会在 100 层时自动报错 |
| `ctx:echo(...)` | 打印到标准输出（自动加换行） | 多参数用空格连接 |
| `ctx:ask(prompt, default?)` | 读取用户输入 | 后台任务（detached）里无输入则返回 default |
| `ctx:env(name)` | 读环境变量 | 不存在返回 nil |
| `ctx:cwd()` | 当前工作目录 | |
| `ctx:os()` | OS 名称 | `"windows"` / `"linux"` / `"darwin"` |

### 例子

```lua
-- 链式调用
ctx:run("setup"):run("build")   -- 但通常写两行更清楚

-- 条件 shell
if ctx:env("CI") then
  ctx:sh("make ci")
else
  ctx:sh("make test")
  ctx:sh("make build")
end

-- 安全的执行（推荐）
ctx:exec("git", {"tag", tag})    -- 不经过 shell，无注入风险
```

### 上下文方法 vs `os.execute`

taskfile 是 **可信代码**（你自己写的 Makefile 替代品），所以你可以直接用 Lua 内置 `os.execute` 做任何事：

```lua
os.execute("rm -rf /")           -- 你自己要负责，tkx 不会保护你
```

但通常用 `ctx:sh` 因为它有错误传播（exit code → Lua error）和 STDIO 继承。

## 11. 常用模式

### 平台分支
```lua
if ctx:os() == "windows" then
  ctx:sh("tasklist /FI \"PID eq 1234\"")
else
  ctx:sh("ps -p 1234")
end
```

### 参数校验（声明式 vs 主动）
声明式（推荐）：
```lua
args = {
  port = { type = "string", required = true, desc = "listen port" },
},
-- tkx 自动校验 --port 是否提供
```

主动校验（args 不写声明）：
```lua
run = function(ctx, args)
  if not args.port then
    error("--port is required")
  end
  if tonumber(args.port) and tonumber(args.port) < 1024 then
    error("--port must be >= 1024")
  end
end
```

### 默认值
```lua
run = function(ctx, args)
  local tag = args.tag or "v0.0.0"
  ctx:sh("git tag " .. tag)         -- 注意：未转义 args.tag 在 git tag 里
end
```

### 安全字符串拼接
`ctx:sh` 把字符串整个塞给 shell。如果用户能控制字符串内容（如 `--tag "v1; rm -rf /"`），就是命令注入。两选一：
- **推荐**：用 `ctx:exec("git", {"tag", tag})`，参数不走 shell
- 或者：shell + 自己转义/校验

### 平台无关路径
```lua
local sep = package.config:sub(1, 1)   -- "/" on Unix, "\" on Windows
-- 或者直接用 ctx:sh("mkdir -p path/parts")  跨平台但依赖 sh
```

### 读取文件
```lua
local f = io.open("/etc/hosts", "r")
if f then
  local content = f:read("*a")
  f:close()
  ctx:echo(content)
end
```

### JSON（不内置）
taskfile 没内置 JSON。如果要解析 JSON 输出：
```lua
-- 需要外部模块；tkx 目前没装 lua-cjson
-- 简单场景：grep + sed 在 shell 里完成
local line = io.popen("jq -r .version package.json 2>/dev/null"):read("*l")
```
**更好的方式**：让脚本输出 `key=value` 行（而不是 JSON），taskfile 用 `string.match` 解析。

## 12. 调试

```lua
-- 打印到 stderr（不影响 CLI 输出流）
io.stderr:write("debug: x=" .. tostring(x) .. "\n")

-- 打印变量类型
ctx:echo(type(x))                    -- "string" / "number" / "table" ...
ctx:echo(#x)                        -- table 长度
ctx:echo(vim.inspect(x))             -- 如果装了 vim 模块（tkx 没默认装）
```

**Taskfile 加载错误**（语法错、文件不存在）显示在 CLI；包含文件名 + 行号：
```
tkx: Taskfile.lua:7: '}' expected (to close '{' at line 3) near 'end'
```

**运行时错误**（命令非零退出、task 抛 error）同样显示行号。

## 13. tkx Lua 与标准 Lua 的差异

| 项 | 标准 Lua | tkx (gopher-lua) |
|---|---|---|
| 版本 | Lua 5.4 | Lua 5.1 |
| 整数/浮点 | 分开 | 都是 double（`42` 和 `42.0` 相同）|
| `goto` | 有 | 有 |
| `continue` | Lua 5.4 有 | 没有，用 `goto` + label 模拟 |
| `string.pack` | 有 | 没有（不用 binary protocol）|
| `coroutine` | 有 | 没有（tkx 不会用到）|
| `require` | 有 | 没有 `package.path`（不要依赖） |
| `io.popen` | 有 | 有，跨平台 |
| `os.execute` | 有 | 有 |
| `os.getenv` | 5.4 | 没有；用 `ctx:env` |
| 整数除法 `//` | 5.3 | 没有，用 `math.floor(a / b)` |

简单说：**别用 5.2+ 的特性**（整数除法、`string.pack`、`continue`）。其余按 5.1 写就行。

## 14. 完整示例

带参数校验、平台分支、链式调用、错误处理：

```lua
local tasks = {
  test = {
    desc = "run tests with optional coverage",
    args = {
      cover = { type = "bool", required = false, desc = "collect coverage" },
      pkg = { type = "string", required = false, desc = "package to test" },
    },
    run = function(ctx, args)
      local pkg = args.pkg or "./..."
      local cmd = "go test "
      if args.cover then
        cmd = cmd .. "-cover "
      end
      cmd = cmd .. pkg
      ctx:sh(cmd)
    end,
  },

  deploy = {
    desc = "build and deploy",
    args = {
      target = { type = "string", required = true, desc = "dev / staging / prod" },
    },
    run = function(ctx, args)
      ctx:run("test", { pkg = "./..." })
      ctx:sh("make build")
      ctx:sh("./scripts/deploy.sh " .. args.target)
      ctx:echo("deployed to " .. args.target)
    end,
  },

  clean = function(ctx)
    ctx:sh("rm -rf dist/ build/")
    ctx:echo("cleaned")
  end,
}
return tasks
```

## 15. 下一步

- 看 `docs/superpowers/specs/2026-08-21-tkx-design.md` 了解设计意图
- 看 `testdata/Taskfile.lua` 看一个可跑的例子
- 看 `README.md` 知道怎么在 `~/.config/taskx/` 放自己的 Taskfile