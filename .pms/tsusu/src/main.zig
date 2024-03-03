const std = @import("std");

const daemon = @import("daemon.zig");

const fs = std.fs;
const os = std.os;
const mem = std.mem;

const Logger = @import("logger.zig").Logger;
const helpers = @import("helpers.zig");
const ProcessStats = @import("process_stats.zig").ProcessStats;

const util = @import("util.zig");
const prettyMemoryUsage = util.prettyMemoryUsage;

pub const Context = struct {
    allocator: std.mem.Allocator,
    args_it: [][]u8,
    tries: usize = 0,

    pub fn init(allocator: std.mem.Allocator, args_it: [][]u8) Context {
        return Context{
            .allocator = allocator,
            .args_it = args_it,
        };
    }

    pub fn checkDaemon(self: *Context) anyerror!std.net.Stream {
        if (self.tries >= 3) return error.SpawnFail;
        self.tries += 1;

        const sock_path = try helpers.getPathFor(self.allocator, .Sock);

        std.debug.print("connecting to socket {s} ...\n", .{sock_path});
        return std.net.connectUnixSocket(sock_path) catch |err| {
            std.debug.print("failed (error: {any}), starting and retrying (try {any})...", .{ err, self.tries });
            try self.spawnDaemon();

            // assuming spawning doesn't take more than a second
            std.time.sleep(1 * std.time.ns_per_s);
            return try self.checkDaemon();
        };
    }

    fn spawnDaemon(self: @This()) !void {
        std.debug.print("Spawning tsusu daemon...\n", .{});
        const data_dir = try helpers.fetchDataDir(self.allocator);
        std.fs.cwd().makePath(data_dir) catch |err| {
            if (err != error.PathAlreadyExists) return err;
        };

        var pid = try std.os.fork();
        if (pid < 0) {
            return error.ForkFail;
        }

        if (pid > 0) {
            return;
        }

        _ = umask(0);

        const daemon_pid = os.linux.getpid();
        const pidpath = try helpers.getPathFor(self.allocator, .Pid);
        const logpath = try helpers.getPathFor(self.allocator, .Log);

        var pidfile = try std.fs.cwd().createFile(pidpath, .{ .truncate = false });
        try pidfile.seekFromEnd(0);

        var stream = pidfile.writer();
        try stream.print("{any}", .{daemon_pid});
        pidfile.close();

        var logfile = try std.fs.cwd().createFile(logpath, .{
            .truncate = false,
        });
        var logstream = logfile.writer();
        var logger = Logger(std.fs.File.Writer).init(logstream, "[d]");

        _ = try setsid();

        try std.os.chdir("/");
        std.os.close(std.os.STDIN_FILENO);
        std.os.close(std.os.STDOUT_FILENO);
        std.os.close(std.os.STDERR_FILENO);

        // redirect stdout and stderr to logfile
        std.os.dup2(logfile.handle, std.os.STDOUT_FILENO) catch |err| {
            logger.info("Failed to dup2 stdout to logfile: {any}", .{err});
        };
        std.os.dup2(logfile.handle, std.os.STDERR_FILENO) catch |err| {
            logger.info("Failed to dup2 stderr to logfile: {any}", .{err});
        };

        // TODO: better way to express that we want to delete pidpath
        // on any scenario. since we exit() at the end, no defer blocks
        // actually run
        daemon.main(&logger) catch |daemon_err| {
            try std.os.unlink(pidpath); // do nothing on errors
            logfile.close();
            return daemon_err;
        };

        try std.os.unlink(pidpath); // do nothing on errors
        logfile.close();
        std.os.exit(0);
    }
};

// TODO upstream mode_t to linux bits x86_64
const mode_t = u32;

fn umask(mode: mode_t) mode_t {
    const rc = os.system.syscall1(os.system.SYS.umask, @as(usize, @bitCast(@as(isize, mode))));
    return @as(mode_t, @intCast(rc));
}

fn setsid() !std.os.pid_t {
    const rc = os.system.syscall0(os.system.SYS.setsid);
    switch (std.os.errno(rc)) {
        std.os.E.SUCCESS => return @as(std.os.pid_t, @intCast(rc)),
        std.os.E.PERM => return error.PermissionFail,
        else => |err| return std.os.unexpectedErrno(err),
    }
}

pub const Mode = enum {
    Destroy,
    Start,
    Stop,
    Help,
    List,
    Noop,
    Logs,
};

fn getMode(mode_arg: []const u8) ?Mode {
         if (std.mem.eql(u8, mode_arg, "noop"))    { return .Noop; }
    else if (std.mem.eql(u8, mode_arg, "destroy")) { return .Destroy; }
    else if (std.mem.eql(u8, mode_arg, "delete"))  { return .Destroy; }
    else if (std.mem.eql(u8, mode_arg, "start"))   { return .Start; }
    else if (std.mem.eql(u8, mode_arg, "stop"))    { return .Stop; }
    else if (std.mem.eql(u8, mode_arg, "help"))    { return .Help; }
    else if (std.mem.eql(u8, mode_arg, "list"))    { return .List; }
    else if (std.mem.eql(u8, mode_arg, "logs"))    { return .Logs; }
    else                                           { return null; }
}

pub fn printServices(msg: []const u8) !void {
    std.debug.print("name | state\t\tpid\tcpu\tmemory\n", .{});
    var it = std.mem.splitScalar(u8, msg, ';');
    while (it.next()) |service_line| {
        if (service_line.len == 0) break;

        var serv_it = std.mem.splitScalar(u8, service_line, ',');
        const name = serv_it.next().?;
        const state_str = serv_it.next().?;
        const state = try std.fmt.parseInt(u8, state_str, 10);

        std.debug.print("{s} |", .{name});

        switch (state) {
        0 => std.debug.print("not running\t\t0\t0%\t0kb", .{}),
        1 => {
            const pid_str = serv_it.next().?;
            const pid = try std.fmt.parseInt(std.os.pid_t, pid_str, 10);

            // we can calculate cpu and ram usage since the service
            // is currently running
            var proc_stats = ProcessStats.init();
            const stats = try proc_stats.fetchAllStats(pid);

            var buffer: [128]u8 = undefined;
            const pretty_memory_usage = try prettyMemoryUsage(&buffer, stats.memory_usage);
            std.debug.print("running\t\t{any}\t{d:.1}%\t{s}", .{ pid, stats.cpu_usage, pretty_memory_usage });
        },
        2 => {
            const exit_code = try std.fmt.parseInt(u32, serv_it.next().?, 10);
            std.debug.print("exited (code {any})\t\t0%\t0kb", .{exit_code});
        },
        3 => {
            const exit_code = try std.fmt.parseInt(u32, serv_it.next().?, 10);
            const remaining_ns = try std.fmt.parseInt(i64, serv_it.next().?, 10);
            std.debug.print("restarting (code {any}, comes in {any}ms)\t\t0%\t0kb", .{
                exit_code,
                @divTrunc(remaining_ns, std.time.ns_per_ms),
            });
        },
        else => unreachable,
        }

        std.debug.print("\n", .{});
    }
}

fn stopCommand(ctx: *Context, in_stream: anytype, out_stream: anytype) !void {
    if (ctx.args_it.len < 2) {
        return error.ExpectedName;
    }

    const name = ctx.args_it[0];
    std.debug.print("stopping '{s}'\n", .{name});

    try out_stream.print("stop;{s}!", .{name});
    const list_msg = try in_stream.readUntilDelimiterAlloc(ctx.allocator, '!', 1024);
    defer ctx.allocator.free(list_msg);

    try printServices(list_msg);
}

fn watchCommand(ctx: *Context, in_stream: anytype, out_stream: anytype) !void {
    if (ctx.args_it.len < 2) {
        std.debug.print("expected name, args: {any}\n", .{ctx.args_it});
        return error.ExpectedName;
    }

    const name = ctx.args_it[0];
    ctx.args_it = ctx.args_it[1..];
    std.debug.print("watching '{s}'\n", .{name});

    try out_stream.print("logs;{s}!", .{name});
    while (true) {
        const msg = try in_stream.readUntilDelimiterAlloc(ctx.allocator, '!', 65535);
        defer ctx.allocator.free(msg);

        // TODO handle when service is stopped

        var it = std.mem.split(u8, msg, ";");
        _ = it.next();
        const service = it.next().?;
        const stream = it.next().?;
        const data = it.next().?;

        std.debug.print("{s} from {s}: {s}", .{ service, stream, data });
    }
}

pub fn main() !void {
    // every time we start, we check if we have a daemon running.
    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();

    var allocator = arena.allocator();

    var args_it = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args_it);
    args_it = args_it[1..];

    if (args_it.len < 1) {
        @panic("expected mode");
    }

    const mode = getMode(args_it[0]) orelse {
        std.debug.print("unknown mode '{s}'\n", .{args_it[0]});
        return;
    };

    // switch for things that don't depend on an existing daemon
    switch (mode) {
    .Destroy => {
        // TODO use sock first (send STOP command), THEN, if it fails, TERM
        const pidpath = try helpers.getPathFor(allocator, .Pid);
        //const sockpath = try helpers.getPathFor(allocator, .Sock);

        var pidfile = std.fs.cwd().openFile(pidpath, .{}) catch |err| {
            std.debug.print("Failed to open PID file ({any}). is the daemon running?\n", .{err});
            return;
        };
        var stream = pidfile.reader();

        const pid_str = try stream.readAllAlloc(allocator, 20);
        defer allocator.free(pid_str);

        const pid_int = std.fmt.parseInt(os.pid_t, pid_str, 10) catch |err| {
            std.debug.print("Failed to parse pid '{s}': {any}\n", .{ pid_str, err });
            return;
        };

        const SIGINT = 15;
        try std.os.kill(pid_int, SIGINT);

        std.debug.print("sent SIGINT to pid {any}\n", .{pid_int});
        return;
    },
    else => {},
    }

    var ctx = Context.init(allocator, args_it);
    const sock = try ctx.checkDaemon();
    defer sock.close();

    var in_stream = sock.reader();
    var out_stream = sock.writer();

    std.debug.print("[c] sock fd to server: {any}\n", .{sock.handle});

    const helo_msg = try in_stream.readUntilDelimiterAlloc(ctx.allocator, '!', 6);
    if (!std.mem.eql(u8, helo_msg, "helo")) {
        std.debug.print("invalid helo, expected helo, got {s}\n", .{helo_msg});
        return error.InvalidHello;
    }

    std.debug.print("[c] first msg (should be helo): {d} '{s}'\n", .{ helo_msg.len, helo_msg });

    switch (mode) {
    .Noop => {},
    .List => {
        _ = try sock.write("list!");

        const msg = try in_stream.readUntilDelimiterAlloc(ctx.allocator, '!', 1024);
        defer ctx.allocator.free(msg);

        if (msg.len == 0) {
            std.debug.print("<no services>\n", .{});
            return;
        }

        try printServices(msg);
    },
    .Start => {
        if (ctx.args_it.len < 2) {
            @panic("expected name");
        }
        ctx.args_it = ctx.args_it[1..];

        try out_stream.print("start;{s}", .{ctx.args_it[0]});

        const path = if (ctx.args_it.len < 2) null else ctx.args_it[1];
        if (path != null) {
            try out_stream.print(";{s}", .{path.?});
        }
        try out_stream.print("!", .{});

        const msg = try in_stream.readUntilDelimiterAlloc(ctx.allocator, '!', 1024);
        defer ctx.allocator.free(msg);

        try printServices(msg);
    },
    .Stop => try stopCommand(&ctx, in_stream, out_stream),
    .Logs => try watchCommand(&ctx, in_stream, out_stream),
    else => std.debug.print("mode {any} is not implemented\n", .{mode}),
    }
}
