const std = @import("std");

pub fn main() !void {
    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();

    var i: usize = 0;
    while (true) : (i += 1) {
        const out = if (i % 2 == 0) stderr else stdout;
        try std.fmt.format(out, "i = {any}\n", .{i});
        std.time.sleep(1 * std.time.ns_per_s);
    }
}
