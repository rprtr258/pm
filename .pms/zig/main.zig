// package main

// import "base:intrinsics"
// import "core:c"

// foreign import libc "system:c"
// foreign libc {
//     environ : [^]cstring
//     strerror :: proc(errno: int) -> cstring ---
// }

// ErrorKill :: enum {
//   OK,
//   EPERM,
//   ESRCH,
//   EINVAL,
// }

// str_error_kill :: proc(err: ErrorKill) -> string {
//   switch err {
//   case ErrorKill.OK:     return "OK"
//   case ErrorKill.EPERM:  return "No permission"
//   case ErrorKill.ESRCH:  return "No such process"
//   case ErrorKill.EINVAL: return "Invalid signal"
//   case:                  unreachable()
//   }
// }

// kill :: proc(pid: os.Pid, sig: int) -> ErrorKill {
//   SYS_kill :: 62
//   err := intrinsics.syscall(SYS_kill, cast(uintptr)pid, cast(uintptr)sig)
//   // for some reason syscall does not write to errno but returns -errno which is
//   // less that -1
//   switch cast(os.Errno)(-err) {
//   case os.ERROR_NONE: return ErrorKill.OK
//   case os.EPERM:      return ErrorKill.EPERM
//   case os.ESRCH:      return ErrorKill.ESRCH
//   case os.EINVAL:     return ErrorKill.EINVAL
//   case:               unreachable()
//   }
// }

// import "core:fmt"
// import "core:os"
// import "core:strconv"

// perror :: proc (errno: os.Errno, msg : string) {
//   fmt.eprint(msg, ": ", strerror(cast(int)errno))
//   fmt.eprintln()
// }

// is_numeric :: proc(str: string) -> bool {
//   for c in str {
//     if c < '0' || c > '9' {
//       return false
//     }
//   }
//   return true
// }

// list_processes :: proc() {
//   proc_dir, errOpen := os.open("/proc")
//   if errOpen != os.ERROR_NONE {
//     perror(errOpen, "Error opening /proc")
//     return
//   }
//   defer os.close(proc_dir)

//   entries, errRead := os.read_dir(proc_dir, 1024)
//   if errRead != os.ERROR_NONE {
//     perror(errRead, "Error reading /proc")
//     return
//   }

//   for entry in entries {
//     if is_numeric(entry.name) {
//       fmt.println(entry.name)
//     }
//   }
// }

// run :: proc(argv: []string) -> int {
//   if len(argv) < 2 {
//     fmt.printf("Usage: %s <list|kill|help> [pid] [signal]\n", argv[0])
//     return 1
//   }

//   switch cmd := argv[1]; cmd {
//   case "list":
//     list_processes()
//   case "kill":
//     if len(argv) < 4 {
//       fmt.eprintf("Usage: %s kill <pid> <signal>\n", argv[0])
//       return 1
//     }

//     if !is_numeric(argv[2]) || !is_numeric(argv[3]) {
//       fmt.eprintf("Error: pid and signal must be integers\n")
//       return 1
//     }

//     signal := strconv.atoi(argv[3])
//     pid := cast(os.Pid)strconv.atoi(argv[2])
//     if err := kill(pid, signal); err != ErrorKill.OK {
//       fmt.printf("Failed to send signal %v to process %v: %s\n", signal, pid, str_error_kill(err))
//       return 1
//     }
//   case "help":
//     fmt.printf(`Usage: %s <list|kill|help> [pid] [signal]
// Commands:
//   list        List all running processes
//   kill <pid>  Kill process with specified PID
//   help        Show this help message
// `, argv[0])
//   case:
//     fmt.eprintf("Error: unknown command %q\n", argv[1])
//     return 1
//   }

//   return 0
// }

const std = @import("std");
// const environ = std.c.environ;
const stdout = std.io.getStdOut().writer();

pub fn main() !void {
    // const envp = environ;
    for (std.os.argv) |arg| {
        try stdout.print("{s}\n", .{arg});
    }
    const environ: [*][*]u8 = std.os.argv.ptr + std.os.argv.len + 1;
    var i: usize = 0;
    while (@as(*allowzero u8, @ptrCast(environ[i])) != @as(*allowzero u8, @ptrFromInt(0))) : (i += 1) {
        const env: [*]u8 = environ[i];
        try stdout.print("{} ", .{i});
        var j: usize = 0;
        while (env[j] != 0) : (j += 1) {
            try stdout.writeByte(env[j]);
        }
        try stdout.writeByte('\n');
    }

    //   os.exit(run(os.args))
}
