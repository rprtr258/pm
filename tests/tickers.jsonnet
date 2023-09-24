[
    {
        name: "tick#%d" % i,
        // command: "tests/tick/main",
        // args: ["--interval", "%(dur)dms" % {dur: i * 10}],
        command: "go",
        args: [
            "run",
            "tick/main.go",
            "--interval",
            "%(dur)dms" % {dur: i * 10},
        ],
        watch: ".*\\.go", 
    } for i in std.range(1, 10)
]
