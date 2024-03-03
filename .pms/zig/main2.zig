const std = @import("std");
const os = std.os;
const out = std.io.getStdOut().writer();

fn is_numeric(str: []const u8) bool {
    for (str) |c| {
        if (c < '0' or c > '9') {
            return false;
        }
    }
    return true;
}

inline fn list_processes() !void {
    var proc_dir = try std.fs.openIterableDirAbsoluteZ("/proc", .{});
    defer proc_dir.close();

    var iter = proc_dir.iterateAssumeFirstIteration();
    while (try iter.next()) |entry| {
        if (is_numeric(entry.name)) {
            try out.print("{s}\n", .{entry.name});
        }
    }
}

const ErrorKill = error{
    InvalidSignal,
    PermissionDenied,
    ProcessNotFound,
};
const ErrorRun = union(enum) {
    Help,
    HelpKill,
    InvalidCommand: []const u8,
    InvalidPid: []const u8,
    InvalidSignal: []const u8,
    Signal: struct{
        Signal: u8,
        Pid: os.pid_t,
        Error: ErrorKill,
    },
};
fn run(argv: [][:0]u8) ?ErrorRun {
    if (argv.len < 2) {
        return .Help;
    }

    if (std.mem.eql(u8, argv[1], "list")) {
        list_processes() catch |err| {
            std.debug.print("Failed to list processes: {}\n", .{err});
        };
    } else if (std.mem.eql(u8, argv[1], "kill")) {
        if (argv.len < 4) {
            return .HelpKill;
        }

        const pid_str = std.mem.sliceTo(argv[2], 0);
        const pid: os.pid_t = std.fmt.parseInt(os.pid_t, pid_str, 10) catch {
            return .{.InvalidPid = pid_str};
        };

        const sig_str = std.mem.sliceTo(argv[3], 0);
        const sig = std.fmt.parseUnsigned(u8, sig_str, 10) catch {
            return .{.InvalidSignal = sig_str};
        };

        _ = blk: {
            switch (os.errno(os.system.kill(pid, sig))) {
                .SUCCESS => break :blk,
                .INVAL => break :blk ErrorKill.InvalidSignal,
                .PERM => break :blk ErrorKill.PermissionDenied,
                .SRCH => break :blk ErrorKill.ProcessNotFound,
                else => unreachable,
            }
        } catch |err| {
            return .{.Signal = .{.Signal = sig, .Pid = pid, .Error = err}};
        };
    } else if (std.mem.eql(u8, argv[1], "help")) {
        return .Help;
    } else {
        return .{.InvalidCommand = argv[1]};
    }

    return null;
}

fn printHelp(cmd: []const u8) !void {
    try out.print(
        \\Usage: {s} COMMAND COMMAND-ARGS
        \\Commands:
        \\  list      List all running processes
        \\  kill PID  Kill process with specified PID
        \\  help      Show this help message
        \\
    , .{cmd});
}

pub fn main() !void {
    const argv = try std.process.argsAlloc(std.heap.page_allocator);
    defer std.process.argsFree(std.heap.page_allocator, argv);

    const cmdname = argv[0];

    if (run(argv)) |err| {
        switch (err) {
        .Help => try printHelp(cmdname),
        .HelpKill => try out.print("Usage: {s} kill <pid> <signal>\n", .{cmdname}),
        .InvalidCommand => |cmd| {
            try out.print("Unknown command: '{s}'\n", .{cmd});
            try printHelp(cmdname);
        },
        .InvalidPid => |pid| try out.print("Invalid pid: {s}\n", .{pid}),
        .InvalidSignal => |sig| try out.print("Invalid signal: {s}\n", .{sig}),
        .Signal => |s| {
            switch (s.Error) {
            ErrorKill.InvalidSignal    => try out.print("Invalid signal: {}\n", .{s.Signal}),
            ErrorKill.PermissionDenied => try out.print("Failed to send signal {} to process {}: {}: permission denied\n", .{s.Signal, s.Pid, s.Error}),
            ErrorKill.ProcessNotFound  => try out.print("Process {} not found\n", .{s.Pid}),
            }
        },
        }
        os.exit(1);
    }
}
