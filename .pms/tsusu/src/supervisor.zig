const std = @import("std");
const daemon = @import("daemon.zig");
const util = @import("util.zig");

const DaemonState = daemon.DaemonState;
const ServiceDecl = daemon.ServiceDecl;
const Service = daemon.Service;
const ServiceStateType = daemon.ServiceStateType;

const ServiceLogger = @import("service_logger.zig").ServiceLogger;

pub const SupervisorContext = struct {
    state: *DaemonState,
    service: *Service,
};

const RECONN_RETRY_CAP = 1000;
const RECONN_RETRY_BASE = 700;

pub fn superviseProcess(ctx: SupervisorContext) !void {
    var state = ctx.state;
    var allocator = state.allocator;
    var service = ctx.service;

    state.logger.info("supervisor start\n", .{});

    var argv = std.ArrayList([]const u8).init(allocator);
    defer argv.deinit();

    var path_it = std.mem.split(u8, ctx.service.cmdline, " ");
    while (path_it.next()) |component| {
        try argv.append(component);
    }

    state.logger.info("sup:{s}: arg0 = {s}\n", .{ ctx.service.name, argv.items[0] });

    var retries: u32 = 0;

    while (!service.stop_flag) {
        state.logger.info("trying to start service '{s}'", .{ctx.service.name});

        var proc = std.ChildProcess.init(argv.items, allocator);

        proc.stdout_behavior = .Pipe;
        proc.stderr_behavior = .Pipe;

        state.logger.info("service '{s}' spawn", .{ctx.service.name});
        try proc.spawn();

        // spawn thread for logging of stderr and stdout
        var logger_pipe = try std.os.pipe();
        defer std.os.close(logger_pipe[0]);
        defer std.os.close(logger_pipe[1]);

        var file = std.fs.File{ .handle = logger_pipe[1] };
        var stream = file.writer();
        var serializer = daemon.MsgSerializer.init(stream);
        _ = serializer;
        _ = std.Thread.spawn(
            .{},
            ServiceLogger.handler,
            .{ServiceLogger.Context{
                .state = ctx.state,
                .service = ctx.service,
                .stdout = proc.stdout.?.handle,
                .stderr = proc.stderr.?.handle,
                .message_fd = logger_pipe[0],
            }},
        ) catch |err| {
            state.logger.info("Failed to start logging thread: {any}", .{err});
            return err;
        };

        state.pushMessage(.{
            .ServiceStarted = .{
                .name = ctx.service.name,
                .pid = proc.id,
                .stdout = proc.stdout.?,
                .stderr = proc.stderr.?,
                .logger_thread = logger_pipe[1],
            },
        }) catch |err| {
            state.logger.info("Failed to send started message to daemon: {any}", .{err});
        };

        state.logger.info("service '{s}' wait", .{ctx.service.name});
        const term_result = try proc.wait();
        state.logger.info("service '{s}' waited result {any}", .{ ctx.service.name, term_result });

        // we don't care about the status of the process if we're here,
        // since it exited already, we must destroy the threads
        // we made for stdout/err
        ServiceLogger.stopLogger(logger_pipe[1]) catch |err| {
            state.logger.info("Failed to signal logger thread to stop: {any}", .{err});
        };

        var exit_code: u32 = undefined;

        switch (term_result) {
        .Exited, .Signal, .Stopped, .Unknown => |term_code| {
            exit_code = term_code;

            // reset retry count when the program exited cleanly
            if (exit_code == 0) retries = 0;

            state.pushMessage(.{
                .ServiceExited = .{ .name = ctx.service.name, .exit_code = exit_code },
            }) catch |err| {
                state.logger.info("Failed to send exited message to daemon: {any}", .{err});
            };
        },
        }

        // if the service is set to stop, we shouldn't try to wait and then
        // stop our loop. stop as soon as possible.

        // this prevents the supervisor thread from living more than it should
        state.logger.info("service '{s}' stop? {any}", .{ ctx.service.name, service.stop_flag });
        if (service.stop_flag) break;

        // calculations are done in the millisecond range
        const seed: u64 = @truncate(@as(u128, @bitCast(std.time.nanoTimestamp())));
        var r = std.rand.DefaultPrng.init(seed);

        const sleep_ms = r.random().uintLessThan(
            u32,
            @min(
                @as(u32, RECONN_RETRY_CAP),
                @as(u32, RECONN_RETRY_BASE * std.math.pow(u32, @as(u32, 2), retries)),
            ),
        );

        const sleep_ns = sleep_ms * std.time.ns_per_ms;
        const clock_ts = util.monotonicRead();

        state.pushMessage(.{
            .ServiceRestarting = .{
                .name = ctx.service.name,
                .exit_code = exit_code,
                .clock_ts_ns = clock_ts,
                .sleep_ns = sleep_ns,
            },
        }) catch |err| {
            state.logger.info("Failed to send restarting message to daemon: {any}", .{err});
        };

        std.debug.print("sleeping '{s}' for {any}ms\n", .{ ctx.service.name, sleep_ms });

        std.time.sleep(sleep_ns);

        std.debug.print("slept '{s}' for {any}ms. stop_flag={any}\n", .{ ctx.service.name, sleep_ms, service.stop_flag });

        retries += 1;
    }
}
