local env = {
  TEST_VARIABLE: "HELLO",
};

[
  {
    name: "echo",
    command: "socat",
    args: ["TCP4-LISTEN:2000,fork", "EXEC:cat"],
  },
  {
    name: "web",
    command: "sh",
    args: ["-c", '"docker build . -t web && docker run web -p 5000:5000"'],
  },
]
