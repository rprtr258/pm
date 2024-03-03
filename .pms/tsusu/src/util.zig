const std = @import("std");

// TODO make this receive a stream and write into it directly instead of
// dealing with memory.
pub fn prettyMemoryUsage(buffer: []u8, kilobytes: u64) ![]const u8 {
    const megabytes = kilobytes / 1024;
    const gigabytes = megabytes / 1024;

    if (kilobytes < 1024) {
        return try std.fmt.bufPrint(buffer, "{d} KB", .{kilobytes});
    } else if (megabytes < 1024) {
        return try std.fmt.bufPrint(buffer, "{d:.2} MB", .{megabytes});
    } else if (gigabytes < 1024) {
        return try std.fmt.bufPrint(buffer, "{d:.2} GB", .{gigabytes});
    } else {
        return try std.fmt.bufPrint(buffer, "how", .{});
    }
}

pub fn read(fd: std.os.fd_t, buf: []u8) !usize {
    // const max_count = switch (std.Target.current.os.tag) {
    //     .linux => 0x7ffff000,
    //     else => std.math.maxInt(isize),
    // };
    const max_count = 0x7ffff000;
    const adjusted_len = @min(max_count, buf.len);

    const rc = std.os.system.read(fd, buf.ptr, adjusted_len);
    switch (std.os.errno(rc)) {
        std.os.E.SUCCESS => return @intCast(rc),
        std.os.E.INVAL   => unreachable,
        std.os.E.FAULT   => unreachable,
        // probably bad to do this mapping
        std.os.E.INTR, std.os.E.AGAIN => return error.WouldBlock,
        std.os.E.BADF      => return error.NotOpenForReading, // Can be a race condition.
        std.os.E.IO        => return error.InputOutput,
        std.os.E.ISDIR     => return error.IsDir,
        std.os.E.NOBUFS    => return error.SystemResources,
        std.os.E.NOMEM     => return error.SystemResources,
        std.os.E.CONNRESET => return error.ConnectionResetByPeer,
        std.os.E.TIMEDOUT  => return error.ConnectionTimedOut,
        else => |err| return std.os.unexpectedErrno(err),
    }
}

/// Wraps a file descriptor with a mutex to prevent
/// data corruption by separate threads, and keeps
/// a `closed` flag to stop threads from trying to
/// operate on something that is closed (that would give EBADF,
/// which is a race condition, aanicking the program)
pub const WrappedWriter = struct {
    file: std.fs.File,
    lock: std.Thread.Mutex,
    closed: bool = false,

    pub fn init(fd: std.os.fd_t) @This() {
        return .{
            .file = std.fs.File{ .handle = fd },
            .lock = std.Thread.Mutex.init(),
        };
    }

    pub fn deinit(self: *@This()) void {
        self.lock.deinit();
    }

    pub fn markClosed(self: *@This()) void {
        const held = self.lock.acquire();
        defer held.release();
        self.closed = true;
    }

    pub const WriterError = std.fs.File.WriteError || error{Closed};
    pub const Writer = std.io.Writer(*@This(), WriterError, write);
    pub fn writer(self: *@This()) Writer {
        return .{ .context = self };
    }

    pub fn write(self: *@This(), bytes: []const u8) std.fs.WriteError!usize {
        const held = self.lock.acquire();
        defer held.release();
        if (self.closed) return error.Closed;
        return try self.file.write(bytes);
    }
};

/// Wraps a file descriptor with a mutex to prevent
/// data corruption by separate threads, and keeps
/// a `closed` flag to stop threads from trying to
/// operate on something that is closed (that would give EBADF,
/// which is a race condition, aanicking the program)
pub const WrappedReader = struct {
    file: std.fs.File,
    lock: std.Thread.Mutex,
    closed: bool = false,

    pub fn init(fd: std.os.fd_t) @This() {
        return .{
            .file = std.fs.File{ .handle = fd },
            .lock = std.Thread.Mutex.init(),
        };
    }

    pub fn deinit(self: *@This()) void {
        self.lock.deinit();
    }

    pub fn markClosed(self: *@This()) void {
        const held = self.lock.acquire();
        defer held.release();
        self.closed = true;
    }

    pub const ReaderError = std.fs.File.ReadError || error{Closed};
    pub const Reader = std.io.Reader(*@This(), ReaderError, @This().read);
    pub fn reader(self: *@This()) Reader {
        return .{ .context = self };
    }

    pub fn read(self: *@This(), data: []u8) ReaderError!usize {
        const held = self.lock.acquire();
        defer held.release();
        if (self.closed) return error.Closed;
        return try self.file.read(data);
    }
};

pub fn monotonicRead() u64 {
    var ts: std.os.timespec = undefined;
    const CLOCK_MONOTONIC = 1;
    std.os.clock_gettime(CLOCK_MONOTONIC, &ts) catch unreachable;
    return @as(u64, @intCast(ts.tv_sec * std.time.ns_per_s + ts.tv_nsec));
}
