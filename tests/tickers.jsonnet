[
    {
        name: "tick#%d" % i,
        command: "tests/tick/main",
        args: ["--interval", "%(dur)dms" % {dur: i * 10}],
    }
    for i in std.range(1, 10)
]
