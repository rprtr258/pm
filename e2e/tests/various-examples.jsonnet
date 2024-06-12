local env = {
  TEST_VARIABLE: "HELLO",
};

# TODO: Make this example dependency-free.
[
  {
    name: "sleeper",
    command: "sleep",
    args: ["infinity"],
  },
  {
    name: "echo",
    command: "socat",
    args: ["TCP4-LISTEN:2000,fork", "EXEC:cat"],
  },
  {
    name: "web",
    command: "sh",
    args: ["-c", '"docker build . -t web && exec docker run web -p 5000:5000"'],
  },
  {
    name: "tick",
    command: "./tests/tick/main",
  },
]
