const std = @import("std");

pub fn Logger(comptime OutStream: type) type {
    return struct {
        stream: OutStream,
        prefix: []const u8,
        lock: std.Thread.Mutex,

        pub fn init(stream: anytype, prefix: []const u8) @This() {
            return .{ .stream = stream, .prefix = prefix, .lock = .{} };
        }

        pub fn deinit(self: *@This()) void {
            self.lock.deinit();
        }

        /// Log a message.
        pub fn info(self: *@This(), comptime fmt: []const u8, args: anytype) void {
            self.lock.lock();
            defer self.lock.unlock();

            const tstamp = std.time.timestamp();
            self.stream.print("{any} {s} ", .{ tstamp, self.prefix }) catch {};
            self.stream.print(fmt, args) catch |err| {
                std.debug.print("error sending line {s} {any}: {any}\n", .{ fmt, args, err });
            };
            _ = self.stream.write("\n") catch {};
        }
    };
}
