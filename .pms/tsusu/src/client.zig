const std = @import("std");

const daemon = @import("daemon.zig");

pub const Client = struct {
    fd: std.os.fd_t,
    lock: std.Thread.Mutex,
    closed: bool = false,

    pub fn init(fd: std.os.fd_t) @This() {
        return .{
            .fd = fd,
            .lock = std.Thread.Mutex{},
        };
    }

    pub fn deinit(self: *@This()) void {
        std.debug.print("client deinit @ {x}\n", .{@intFromPtr(self)});
        std.os.close(self.fd);
    }

    pub fn close(self: *@This()) void {
        self.lock.lock();
        defer self.lock.unlock();
        self.closed = true;
    }

    pub fn stream(self: *@This()) std.fs.File.Writer {
        var file = std.fs.File{ .handle = self.fd };
        return file.writer();
    }

    pub fn write(self: *@This(), data: []const u8) !void {
        const held = self.lock.acquire();
        defer held.release();
        if (self.closed) return error.Closed;
        return try self.stream().write(data);
    }

    pub fn print(self: *@This(), comptime fmt: []const u8, args: anytype) !void {
        self.lock.lock();
        defer self.lock.unlock();
        if (self.closed) return error.Closed;
        return try self.stream().print(fmt, args);
    }
};
