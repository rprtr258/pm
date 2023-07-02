[
  {
    name: "qmen24-" + std.extVar("now"),
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
    } + std.native("dotenv")(".test.env")
  },
]
