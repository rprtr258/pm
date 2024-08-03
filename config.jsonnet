local dotenv = std.native("dotenv");
local now = std.extVar("now");

[
  {
    name: "qmen24-" + now,
    command: "sleep",
    args: [10],
  },
  {
    name: "xdd",
    command: "sleep",
    args: ["1000"],
  },
  {
    name: "ls2",
    command: "ls",
    cwd: "..",
  },
  {
    command: "sleep",
    args: [20],
  },
  {
    command: "pwd",
  },
  {
    name: "hello-world",
    command: "go",
    args: ["run", "tests/main.go"],
  },
] + [
  {
    name: "http-hello-server",
    command: "go",
    args: ["run", "tests/hello-http/main.go"],
  },
  {
    name: "test-env",
    command: "rwenv",
    env: {
      TEST_VAR: "test1",
      FROMCONFIG: "fromconfig456",
    } + dotenv(".test.env")
  },
  {
    name: "web",
    cwd: "./tests/example-http",
    command: "sh",
    args: ["-c", |||
      docker build -t web . &&
      exec docker run -p 44224:44224 -e PORT=44224 --env-file ./env web
    |||],
  },
  {
    name: "hang",
    cwd: "./tests/hang",
    command: "go",
    args: ["run", "main.go"],
  },
] + [
  {
    name: "tick#%d" % i,
    // command: "tests/tick/main",
    // args: ["--interval", "%(dur)dms" % {dur: i * 10}],
    command: "go",
    cwd: "tests",
    args: [
      "run",
      "tick/main.go",
      "%(dur)dms" % {dur: i * 10},
    ],
    tags: ["ticker"],
    watch: ".*\\.go",
  } for i in std.range(1, 10)
]
