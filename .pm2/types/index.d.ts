/**
 * Either connects to a running pm2 daemon (“God”) or launches and daemonizes one.
 * Once launched, the pm2 process will keep running after the script exits.
 * @param errback - Called when finished connecting to or launching the pm2 daemon process.
 */
export function connect(errback: ErrCallback): void;
/**
 * Either connects to a running pm2 daemon (“God”) or launches and daemonizes one.
 * Once launched, the pm2 process will keep running after the script exits.
 * @param noDaemonMode - (Default: false) If true is passed for the first argument
 * pm2 will not be run as a daemon and will die when the related script exits.
 * By default, pm2 stays alive after your script exits.
 * If pm2 is already running, your script will link to the existing daemon but will die once your process exits.
 * @param errback - Called when finished connecting to or launching the pm2 daemon process.
 */
export function connect(noDaemonMode:boolean, errback: ErrCallback): void;

// Disconnects from the pm2 daemon.
export function disconnect(): void;

/**
 * Stops a process but leaves the process meta-data in pm2’s list
 * @param process - Can either be the name as given in the pm2.start options,
 * a process id, or the string “all” to indicate that all scripts should be restarted.
 * @param errback - called when the process is stopped
 */
export function stop(process: string|number, errback: ErrProcCallback): void;

/**
 * Stops and restarts the process.
 * @param process - Can either be the name as given in the pm2.start options,
 * a process id, or the string “all” to indicate that all scripts should be restarted.
 * @param errback - called when the process is restarted
 */
export function restart(process: string|number, errback: ErrProcCallback): void;

/**
 * Stops the process and removes it from pm2’s list.
 * The process will no longer be accessible by its name
 * @param process - Can either be the name as given in the pm2.start options,
 * a process id, or the string “all” to indicate that all scripts should be restarted.
 * @param errback - called when the process is deleted
 */
declare function del(process: string|number, errback: ErrProcCallback): void;
// have to use this construct because `delete` is a reserved word
export {del as delete};

/**
 * Zero-downtime rolling restart. At least one process will be kept running at
 * all times as each instance is restarted individually.
 * Only works for scripts started in cluster mode.
 * @param process - Can either be the name as given in the pm2.start options,
 * a process id, or the string “all” to indicate that all scripts should be restarted.
 * @param errback - called when the process is reloaded
 */
export function reload(process: string|number, errback: ErrProcCallback): void;

/**
 * Zero-downtime rolling restart. At least one process will be kept running at
 * all times as each instance is restarted individually.
 * Only works for scripts started in cluster mode.
 * @param process - Can either be the name as given in the pm2.start options,
 * a process id, or the string “all” to indicate that all scripts should be restarted.
 * @param options - An object containing configuration
 * @param options.updateEnv - (Default: false) If true is passed in, pm2 will reload it’s
 * environment from process.env before reloading your process.
 * @param errback - called when the process is reloaded
 */
export function reload(process: string|number, options: ReloadOptions, errback: ErrProcCallback): void;

/**
 * Kills the pm2 daemon (same as pm2 kill). Note that when the daemon is killed, all its
 * processes are also killed. Also note that you still have to explicitly disconnect
 * from the daemon even after you kill it.
 */
export function killDaemon(errback: ErrProcDescCallback): void;

/**
 * Returns various information about a process: eg what stdout/stderr and pid files are used.
 * @param process - Can either be the name as given in the pm2.start options,
 * a process id, or the string “all” to indicate that all scripts should be restarted.
 */
export function describe(process: string|number, errback: ErrProcDescsCallback): void;

// Gets the list of running processes being managed by pm2.
export function list(errback: ErrProcDescsCallback): void;

/**
 * Writes the process list to a json file at the path in the DUMP_FILE_PATH environment variable
 * (“~/.pm2/dump.pm2” by default).
 */
export function dump(errback: ErrResultCallback): void;

/**
 * Flushes the logs.
 * @param process - Can either be the name as given in the pm2.start options,
 * a process id, or the string “all” to indicate that all scripts should be restarted.
 */
export function flush(process: number|string, errback: ErrResultCallback): void;

export function dump(errback: ErrResultCallback): void;

/**
 * Rotates the log files. The new log file will have a higher number
 * in it (the default format being ${process.name}-${out|err}-${number}.log).
 */
export function reloadLogs(errback: ErrResultCallback): void;

/**
 * Opens a message bus.
 * @param errback The bus will be an Axon Sub Emitter object used to listen to and send events.
 */
export function launchBus(errback: ErrBusCallback): void;

/**
 * @param process - Can either be the name as given in the pm2.start options,
 * a process id, or the string “all” to indicate that all scripts should be restarted.
 */
export function sendSignalToProcessName(signal:string|number, process: number|string, errback: ErrResultCallback): void;

// Registers the script as a process that will start on machine boot. The current process list will be dumped and saved for resurrection on reboot.
export function startup(platform: Platform, errback: ErrResultCallback): void;

// Send an set of data as object to a specific process
export function sendDataToProcessId(proc_id: number, packet: object, cb: ErrResultCallback): void;

// An object with information about the process.
export interface ProcessDescription {
  // The name given in the original start command.
  name?: string;
  // The pid of the process.
  pid?: number;
  // The pid for the pm2 God daemon process.
  pm_id?: number;
  monit?: Monit;
  // The list of path variables in the process’s environment
  pm2_env?: Pm2Env;
}

interface Monit {
  // The number of bytes the process is using.
  memory?: number;
  // The percent of CPU being used by the process at the moment.
  cpu?: number;
}

// The list of path variables in the process’s environment
interface Pm2Env {
  // The working directory of the process.
  pm_cwd?: string;
  // The stdout log file path.
  pm_out_log_path?: string;
  // The stderr log file path.
  pm_err_log_path?: string;
  // The interpreter used.
  exec_interpreter?: string;
  // The uptime of the process.
  pm_uptime?: number;
  // The number of unstable restarts the process has been through.
  unstable_restarts?: number;
  restart_time?: number;
  status?: ProcessStatus;
  // The number of running instances.
  instances?: number | 'max';
  // The path of the script being run in this process.
  pm_exec_path?: string;
}

interface ReloadOptions {
  /**
   * (Default: false) If true is passed in, pm2 will reload it’s environment from process.env 
   * before reloading your process.
   */
  updateEnv?: boolean;
}

type ProcessStatus = 'online' | 'stopping' | 'stopped' | 'launching' | 'errored' | 'one-launch-status';
type Platform = 'ubuntu' | 'centos' | 'redhat' | 'gentoo' | 'systemd' | 'darwin' | 'amazon';

type ErrCallback = (err: Error) => void;
type ErrProcDescCallback = (err: Error, processDescription: ProcessDescription) => void;
type ErrProcDescsCallback = (err: Error, processDescriptionList: ProcessDescription[]) => void;
type ErrResultCallback = (err: Error, result: any) => void;
type ErrBusCallback = (err: Error, bus: any) => void;
