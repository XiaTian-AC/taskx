local tasks = {
  echo = function(ctx)
    ctx:echo("hello from echo task")
  end,

  quick = function(ctx)
    ctx:sh("echo line1")
    ctx:sh("echo line2")
    ctx:sh("echo line3")
  end,

  longrun = function(ctx)
    ctx:sh("echo will-sleep")
    if ctx:os() == "windows" then
      ctx:sh("Start-Sleep -Seconds 30")
    else
      ctx:sh("sleep 30")
    end
  end,

  tagged = {
    desc = "echo the tag argument",
    args = {
      tag = { type = "string", required = true, desc = "a version tag" },
    },
    run = function(ctx, args)
      ctx:echo("tag=" .. args.tag)
    end,
  },
}
return tasks
