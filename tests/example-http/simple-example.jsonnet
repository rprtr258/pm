[{
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
}]
