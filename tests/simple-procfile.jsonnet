# TODO: Make this example dependency-free.
[
    {
        name: "echo",
        command: "socat",
        args: ["TCP4-LISTEN:2000,fork", "EXEC:cat"],
    },
    {
        name: "tick",
        command: "tick",
    },
]
