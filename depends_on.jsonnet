[
  {
    name: "touch",
    command: "touch",
    args: ["tralala"],
  },
  {
    name: "info",
    command: "ls",
    args: ["-l", "tralala"],
    depends_on: ["touch"],
  },
  {
    name: "rm",
    command: "rm",
    args: ["tralala"],
    depends_on: ["info"],
  },
]
