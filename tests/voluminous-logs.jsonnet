[
    {
        name: "tick#%d" % i,
        command: "tick",
        args: ["--interval", "%(dur)dms" % {dur: i * 10}],
    }
    for i in std.range(1, 10)
]
