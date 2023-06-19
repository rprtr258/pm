[
  {
    name: "echo",
      command   : "nc",
      args:  ["-l", "-p", "44222"],
  },
  {
    name: "echo-short",
    command: "nc",
    args: ["-l", "-p", "44223"],
  },
  {
    name: "web",
    command: "sh",
    args: ["-c", |||
      docker build -t web %(build)s
      docker run -p %(ports)s -e PORT=44224 --env-file %(env_file)s web
    ||| % {
      build: ".",
      ports: std.join("-p ", ["44224:44224"]),
      environment: {
        PORT: "44224",
      },
      env_file: "./env",
    }],
  },
]
