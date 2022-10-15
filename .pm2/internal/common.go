// Common Utilities ONLY USED IN ->CLI<-
package internal

import (
	// var fs        = require('fs');
	// var util      = require('util');
	// var chalk     = require('chalk');
	// var fclone    = require('fclone');
	// var semver    = require('semver');
	// var dayjs     = require('dayjs');
	// var execSync  = require('child_process').execSync;

	"os/user"
	"path"
	"strings"
	// var isBinary  = require('./tools/isbinaryfile.js');
	// var extItps   = require('./API/interpreter.json');
	// var Config    = require('./tools/Config');
	// var pkg       = require('../package.json');
	// var which     = require('./tools/which.js');
)

// function homedir() {
//   var env = process.env;
//   var home = env.HOME;
//   var user = env.LOGNAME || env.USER || env.LNAME || env.USERNAME;

//   if (process.platform == 'win32') {
//     return env.USERPROFILE || env.HOMEDRIVE + env.HOMEPATH || home || null;
//   }

//   if (process.platform == 'darwin') {
//     return home || (user ? '/Users/' + user : null);
//   }

//   if (process.platform == 'linux') {
//     return home || (process.getuid() == 0 ? '/root' : (user ? '/home/' + user : null));
//   }

//   return home || null;
// }

// function resolveHome(filepath) {
//   if (filepath[0] == '~') {
//     return path.join(homedir(), filepath.slice(1));
//   }
//   return filepath;
// }

// Common.determineSilentCLI = function() {
//   // pm2 should ignore -s --silent -v if they are after '--'
//   var variadicArgsDashesPos = process.argv.indexOf('--');
//   var s1opt = process.argv.indexOf('--silent')
//   var s2opt = process.argv.indexOf('-s')

//   if (process.env.PM2_SILENT || (variadicArgsDashesPos > -1 &&
//        (s1opt != -1 && s1opt < variadicArgsDashesPos) &&
//        (s2opt != -1 != s2opt < variadicArgsDashesPos)) ||
//       (variadicArgsDashesPos == -1 && (s1opt > -1 || s2opt > -1))) {
//     for (var key in console){
//       var code = key.charCodeAt(0);
//       if (code >= 97 && code <= 122){
//         console[key] = function(){};
//       }
//     }
//     process.env.PM2_DISCRETE_MODE = true;
//   }
// }

// Common.printVersion = function() {
//   var variadicArgsDashesPos = process.argv.indexOf('--');

//   if (process.argv.indexOf('-v') > -1 && process.argv.indexOf('-v') < variadicArgsDashesPos) {
//     console.log(pkg.version);
//     process.exit(0);
//   }
// }

// Common.lockReload = function() {
//   try {
//     var t1 = fs.readFileSync(cst.PM2_RELOAD_LOCKFILE).toString();

//     // Check if content and if time < 30 return locked
//     // Else if content detected (lock file staled), allow and rewritte
//     if (t1 && t1 != '') {
//       var diff = dayjs().diff(parseInt(t1));
//       if (diff < cst.RELOAD_LOCK_TIMEOUT)
//         return diff;
//     }
//   } catch(e) {}

//   try {
//     // Write latest timestamp
//     fs.writeFileSync(cst.PM2_RELOAD_LOCKFILE, dayjs().valueOf().toString());
//     return 0;
//   } catch(e) {
//     console.error(e.message || e);
//   }
// };

// Common.unlockReload = function() {
//   try {
//     fs.writeFileSync(cst.PM2_RELOAD_LOCKFILE, '');
//   } catch(e) {
//     console.error(e.message || e);
//   }
// };

// /**
//  * Resolve app paths and replace missing values with defaults.
//  * @method prepareAppConf
//  * @param app {Object}
//  * @param {} cwd
//  * @param {} outputter
//  * @return app
//  */
// Common.prepareAppConf = function(opts, app) {
//   /**
//    * Minimum validation
//    */
//   if (!app.script)
//     return new Error('No script path - aborting');

//   var cwd = null;

//   if (app.cwd) {
//     cwd = path.resolve(app.cwd);
//     process.env.PWD = app.cwd;
//   }

//   if (!app.node_args) {
//     app.node_args = [];
//   }

//   if (app.port && app.env) {
//     app.env.PORT = app.port;
//   }

//   // CWD option resolving
//   cwd && (cwd[0] != '/') && (cwd = path.resolve(process.cwd(), cwd));
//   cwd = cwd || opts.cwd;

//   // Full path script resolution
//   app.pm_exec_path = path.resolve(cwd, app.script);

//   // If script does not exist after resolution
//   if (!fs.existsSync(app.pm_exec_path)) {
//     var ckd;
//     // Try resolve command available in $PATH
//     if ((ckd = which(app.script))) {
//       if (typeof(ckd) !== 'string')
//         ckd = ckd.toString();
//       app.pm_exec_path = ckd;
//     }
//     else
//       // Throw critical error
//       return new Error(`Script not found: ${app.pm_exec_path}`);
//   }

//   /**
//    * Auto detect .map file and enable source map support automatically
//    */
//   if (app.disable_source_map_support != true) {
//     try {
//       fs.accessSync(app.pm_exec_path + '.map', fs.R_OK);
//       app.source_map_support = true;
//     } catch(e) {}
//     delete app.disable_source_map_support;
//   }

//   delete app.script;

//   // Set current env by first adding the process environment and then extending/replacing it
//   // with env specified on command-line or JSON file.

//   var env = {};

//   /**
//    * Do not copy internal pm2 environment variables if acting on process
//    * is made from a programmatic script started by PM2 or if a pm_id is present in env
//    */
//   if (cst.PM2_PROGRAMMATIC || process.env.pm_id)
//     Common.safeExtend(env, process.env);
//   else
//     env = process.env;

//   function filterEnv (envObj) {
//     if (app.filter_env == true)
//       return {}

//     if (typeof app.filter_env == 'string') {
//       delete envObj[app.filter_env]
//       return envObj
//     }

//     var new_env = {};
//     var allowedKeys = app.filter_env.reduce((acc, current) =>
//                                             acc.filter( item => !item.includes(current)), Object.keys(envObj))
//     allowedKeys.forEach( key => new_env[key] = envObj[key]);
//     return new_env
//   }

//   app.env = [
//     {}, (app.filter_env && app.filter_env.length > 0) ? filterEnv(process.env) : env, app.env || {}
//   ].reduce(function(e1, e2){
//     return Object.assign(e1, e2);
//   });

//   app.pm_cwd = cwd;
//   // Interpreter
//   try {
//     Common.sink.resolveInterpreter(app);
//   } catch(e) {
//     return e
//   }

//   // Exec mode and cluster stuff
//   Common.sink.determineExecMode(app);

//   /**
//    * Scary
//    */
//   var formated_app_name = app.name.replace(/[^a-zA-Z0-9\\.\\-]/g, '-');

//   ['log', 'out', 'error', 'pid'].forEach(function(f){
//     var af = app[f + '_file'], ps, ext = (f == 'pid' ? 'pid':'log'), isStd = !~['log', 'pid'].indexOf(f);
//     if (af) af = resolveHome(af);

//     if ((f == 'log' && typeof af == 'boolean' && af) || (f != 'log' && !af)) {
//       ps = [cst['DEFAULT_' + ext.toUpperCase() + '_PATH'], formated_app_name + (isStd ? '-' + f : '') + '.' + ext];
//     } else if ((f != 'log' || (f == 'log' && af)) && af !== 'NULL' && af !== '/dev/null') {
//       ps = [cwd, af];

//       var dir = path.dirname(path.resolve(cwd, af));
//       if (!fs.existsSync(dir)) {
//         Common.printError(cst.PREFIX_MSG_WARNING + 'Folder does not exist: ' + dir);
//         Common.printOut(cst.PREFIX_MSG + 'Creating folder: ' + dir);
//         try {
//           require('mkdirp').sync(dir);
//         } catch (err) {
//           Common.printError(cst.PREFIX_MSG_ERR + 'Could not create folder: ' + path.dirname(af));
//           throw new Error('Could not create folder');
//         }
//       }

//     }
//     // PM2 paths
//     if (af !== 'NULL' && af !== '/dev/null') {
//       ps && (app['pm_' + (isStd ? f.substr(0, 3) + '_' : '') + ext + '_path'] = path.resolve.apply(null, ps));
//     } else if (path.sep == '\\') {
//       app['pm_' + (isStd ? f.substr(0, 3) + '_' : '') + ext + '_path'] = '\\\\.\\NUL';
//     } else {
//       app['pm_' + (isStd ? f.substr(0, 3) + '_' : '') + ext + '_path'] = '/dev/null';
//     }
//     delete app[f + '_file'];
//   });

//   return app;
// };

// Check if filename is a configuration file. Returns null if not conf file, json or yaml otherwise.
func isConfigFile(filename string) Optional[string] {
	switch path.Ext(filename) {
	case ".json":
		return Valid("json")
	case ".yml", ".yaml":
		return Valid("yaml")
	default:
		return Invalid[string]()
	}
}

// /**
//  * Parses a config file like ecosystem.config.js. Supported formats: JS, JSON, JSON5, YAML.
//  * @param {string} confString  contents of the config file
//  * @param {string} filename    path to the config file
//  * @return {Object} config object
//  */
// Common.parseConfig = function(confObj, filename) {
//   var yamljs = require('yamljs');
//   var vm     = require('vm');

//   if (!filename ||
//       filename == 'pipe' ||
//       filename == 'none' ||
//       filename.indexOf('.json') > -1) {
//     var code = '(' + confObj + ')';
//     var sandbox = {};

//     return vm.runInThisContext(code, sandbox, {
//       filename: path.resolve(filename),
//       displayErrors: false,
//       timeout: 1000
//     });
//   }
//   else if (filename.indexOf('.yml') > -1 ||
//            filename.indexOf('.yaml') > -1) {
//     return yamljs.parse(confObj.toString());
//   }
//   else if (filename.indexOf('.config.js') > -1 || filename.indexOf('.config.cjs') > -1 || filename.indexOf('.config.mjs') > -1) {
//     var confPath = require.resolve(path.resolve(filename));
//     delete require.cache[confPath];
//     return require(confPath);
//   }
// };

// Common.sink = {};

func determineCron(app any) error {
	//   if app.cron_restart == 0 || app.cron_restart == '0' {
	//     Common.printOut(cst.PREFIX_MSG + "disabling cron restart");
	//     return
	//   }

	//   if (app.cron_restart) {
	//     const Croner = require("croner");

	//	  try {
	//	    Common.printOut(cst.PREFIX_MSG + "cron restart at " + app.cron_restart);
	//	    Croner(app.cron_restart);
	//	  } catch(ex) {
	//	    return new Error(`Cron pattern error: ${ex.message}`);
	//	  }
	//	}
	return nil
}

// Handle alias (fork <=> fork_mode, cluster <=> cluster_mode)
// Common.sink.determineExecMode = function(app) {
//   if (app.exec_mode)
//     app.exec_mode = app.exec_mode.replace(/^(fork|cluster)$/, '$1_mode');

//   /**
//    * Here we put the default exec mode
//    */
//   if (!app.exec_mode &&
//       (app.instances >= 1 || app.instances == 0 || app.instances == -1) &&
//       app.exec_interpreter.indexOf('node') > -1) {
//     app.exec_mode = 'cluster_mode';
//   } else if (!app.exec_mode) {
//     app.exec_mode = 'fork_mode';
//   }
//   if (typeof app.instances == 'undefined')
//     app.instances = 1;
// };

// var resolveNodeInterpreter = function(app) {
//   if (app.exec_mode && app.exec_mode.indexOf('cluster') > -1) {
//     Common.printError(cst.PREFIX_MSG_WARNING + chalk.bold.yellow('Choosing the Node.js version in cluster mode is not supported'));
//     return false;
//   }

//   var nvm_path = cst.IS_WINDOWS ? process.env.NVM_HOME : process.env.NVM_DIR;
//   if (!nvm_path) {
//     Common.printError(cst.PREFIX_MSG_ERR + chalk.red('NVM is not available in PATH'));
//     Common.printError(cst.PREFIX_MSG_ERR + chalk.red('Fallback to node in PATH'));
//     var msg = cst.IS_WINDOWS
//       ? 'https://github.com/coreybutler/nvm-windows/releases/'
//       : '$ curl https://raw.githubusercontent.com/creationix/nvm/master/install.sh | bash';
//     Common.printOut(cst.PREFIX_MSG_ERR + chalk.bold('Install NVM:\n' + msg));
//   }
//   else {
//     var node_version  = app.exec_interpreter.split('@')[1];
//     var path_to_node  = cst.IS_WINDOWS
//       ? '/v' + node_version + '/node.exe'
//       : semver.satisfies(node_version, '>= 0.12.0')
//           ? '/versions/node/v' + node_version + '/bin/node'
//           : '/v' + node_version + '/bin/node';
//     var nvm_node_path  = path.join(nvm_path, path_to_node);
//     try {
//       fs.accessSync(nvm_node_path);
//     } catch(e) {
//       Common.printOut(cst.PREFIX_MSG + 'Installing Node v%s', node_version);
//       var nvm_bin = path.join(nvm_path, 'nvm.' + (cst.IS_WINDOWS ? 'exe' : 'sh'));
//       var nvm_cmd = cst.IS_WINDOWS
//         ? nvm_bin + ' install ' + node_version
//         : '. ' + nvm_bin + ' ; nvm install ' + node_version;

//       Common.printOut(cst.PREFIX_MSG + 'Executing: %s', nvm_cmd);

//       execSync(nvm_cmd, {
//         cwd: path.resolve(process.cwd()),
//         env: process.env,
//         maxBuffer: 20 * 1024 * 1024
//       });

//       // in order to support both arch, nvm for Windows renames 'node.exe' to:
//       // 'node32.exe' for x32 arch
//       // 'node64.exe' for x64 arch
//       if (cst.IS_WINDOWS)
//         nvm_node_path = nvm_node_path.replace(/node/, 'node' + process.arch.slice(1))
//     }

//     Common.printOut(cst.PREFIX_MSG + chalk.green.bold('Setting Node to v%s (path=%s)'),
//                     node_version,
//                     nvm_node_path);

//     app.exec_interpreter = nvm_node_path;
//   }
// };

// Resolve interpreter
// Common.sink.resolveInterpreter = function(app) {
//   var noInterpreter = !app.exec_interpreter;
//   var extName = path.extname(app.pm_exec_path);
//   var betterInterpreter = extItps[extName];

//   // No interpreter defined and correspondance in schema hashmap
//   if (noInterpreter && betterInterpreter) {
//     app.exec_interpreter = betterInterpreter;
//   }
//   // Else if no Interpreter detect if process is binary
//   else if (noInterpreter)
//     app.exec_interpreter = isBinary(app.pm_exec_path) ? 'none' : 'node';
//   else if (app.exec_interpreter.indexOf('node@') > -1)
//     resolveNodeInterpreter(app);

//   if (app.exec_interpreter.indexOf('python') > -1)
//     app.env.PYTHONUNBUFFERED = '1'

//   /**
//    * Specific installed JS transpilers
//    */
//   if (app.exec_interpreter == 'ts-node') {
//     app.exec_interpreter = path.resolve(__dirname, '../node_modules/.bin/ts-node');
//   }

//   if (app.exec_interpreter == 'lsc') {
//     app.exec_interpreter = path.resolve(__dirname, '../node_modules/.bin/lsc');
//   }

//   if (app.exec_interpreter == 'coffee') {
//     app.exec_interpreter = path.resolve(__dirname, '../node_modules/.bin/coffee');
//   }

//   if (app.exec_interpreter != 'none' && which(app.exec_interpreter) == null) {
//     // If node is not present
//     if (app.exec_interpreter == 'node') {
//       Common.warn(`Using builtin node.js version on version ${process.version}`)
//       app.exec_interpreter = cst.BUILTIN_NODE_PATH
//     }
//     else
//       throw new Error(`Interpreter ${app.exec_interpreter} is NOT AVAILABLE in PATH. (type 'which ${app.exec_interpreter}' to double check.)`)
//   }

//   return app;
// };

// /**
//  * Raw extend
//  */
// Common.extend = function(destination, source) {
//   if (typeof destination !== 'object') {
//     destination = {};
//   }
//   if (!source || typeof source !== 'object') {
//     return destination;
//   }

//   Object.keys(source).forEach(function(new_key) {
//     if (source[new_key] != '[object Object]')
//       destination[new_key] = source[new_key];
//   });

//   return destination;
// };

// /**
//  * This is useful when starting script programmatically
//  */
// Common.safeExtend = function(origin, add){
//   if (!add || typeof add != 'object') return origin;

//   //Ignore PM2's set environment variables from the nested env
//   var keysToIgnore = ['name', 'exec_mode', 'env', 'args', 'pm_cwd', 'exec_interpreter', 'pm_exec_path', 'node_args', 'pm_out_log_path', 'pm_err_log_path', 'pm_pid_path', 'pm_id', 'status', 'pm_uptime', 'created_at', 'windowsHide', 'username', 'merge_logs', 'kill_retry_time', 'prev_restart_delay', 'instance_var', 'unstable_restarts', 'restart_time', 'axm_actions', 'pmx_module', 'command', 'watch', 'filter_env', 'versioning', 'vizion_runing', 'MODULE_DEBUG', 'pmx', 'axm_options', 'created_at', 'watch', 'vizion', 'axm_dynamic', 'axm_monitor', 'instances', 'automation', 'autorestart', 'unstable_restart', 'treekill', 'exit_code', 'vizion'];

//   var keys = Object.keys(add);
//   var i = keys.length;
//   while (i--) {
//   	//Only copy stuff into the env that we don't have already.
//   	if(keysToIgnore.indexOf(keys[i]) == -1 && add[keys[i]] != '[object Object]')
//       origin[keys[i]] = add[keys[i]];
//   }
//   return origin;
// };

// /**
//  * Extend the app.env object of with the properties taken from the
//  * app.env_[envName] and deploy configuration.
//  * Also update current json attributes
//  *
//  * Used only for Configuration file processing
//  *
//  * @param {Object} app The app object.
//  * @param {string} envName The given environment name.
//  * @param {Object} deployConf Deployment configuration object (from JSON file or whatever).
//  * @returns {Object} The app.env variables object.
//  */
// Common.mergeEnvironmentVariables = function(app_env, env_name, deploy_conf) {
//   var app = fclone(app_env);

//   var new_conf = {
//     env : {}
//   }

//   // Stringify possible object
//   for (var key in app.env) {
//     if (typeof app.env[key] == 'object') {
//       app.env[key] = JSON.stringify(app.env[key]);
//     }
//   }

//   /**
//    * Extra configuration update
//    */
//   Object.assign(new_conf, app);

//   if (env_name) {
//     // First merge variables from deploy.production.env object as least priority.
//     if (deploy_conf && deploy_conf[env_name] && deploy_conf[env_name]['env']) {
//       Object.assign(new_conf.env, deploy_conf[env_name]['env']);
//     }

//     Object.assign(new_conf.env, app.env);

//     // Then, last and highest priority, merge the app.env_production object.
//     if ('env_' + env_name in app) {
//       Object.assign(new_conf.env, app['env_' + env_name]);
//     }
//     else {
//       Common.printOut(cst.PREFIX_MSG_WARNING + chalk.bold('Environment [%s] is not defined in process file'), env_name);
//     }
//   }

//   delete new_conf.exec_mode

//   var res = {
//     current_conf: {}
//   }

//   Object.assign(res, new_conf.env);
//   Object.assign(res.current_conf, new_conf);

//   // #2541 force resolution of node interpreter
//   if (app.exec_interpreter &&
//       app.exec_interpreter.indexOf('@') > -1) {
//     resolveNodeInterpreter(app);
//     res.current_conf.exec_interpreter = app.exec_interpreter
//   }

//   return res
// }

// /**
//  * This function will resolve paths, option and environment
//  * CALLED before 'prepare' God call (=> PROCESS INITIALIZATION)
//  * @method resolveAppAttributes
//  * @param {Object} opts
//  * @param {Object} opts.cwd
//  * @param {Object} opts.pm2_home
//  * @param {Object} appConf application configuration
//  * @return app
//  */
// Common.resolveAppAttributes = function(opts, conf) {
//   var conf_copy = fclone(conf);

//   var app = Common.prepareAppConf(opts, conf_copy);
//   if (app instanceof Error) {
//     throw new Error(app.message);
//   }
//   return app;
// }

type RunConfig struct {
	// TODO: remove everything about "cluster"
	// exec_mode       Optional[string] // "fork" or "cluster"
	// TODO: separate "aboba 1 2 | xdd 3 4 | abigus 5 > bobus < avtobus" from just "<cmd> <args...>"
	command string
	args    []string
	// env             map[string]string
	// execute_command bool
	// username        string
	// name            Optional[string]
	namespace string
	// log_date_format Optional[string]
	// uid             Optional[int]
	// gid             int
	// user            Optional[string]
	// deep_monitoring bool
	// pmx             bool
	// disable_trace   bool
	// trace           bool
	// merge_logs      bool
	// cron_restart    bool
}

// TODO: parse array of raw configs into internal structs
// Verify configurations. Called on EVERY Operation (start/restart/reload/stop...)
func validateRunConfig(appConfs []RunConfig) error {
	return nil
	// verifiedConf := []RunConfig{}

	// for _, app := range appConfs {
	// 	// TODO: remove mutation
	// 	if app.exec_mode.Valid {
	// 		app.exec_mode.Value += "_mode"
	// 	}

	// 	if app.command.Valid {
	// 		app.script = app.command
	// 	}

	// 	// Render an app name if not existing.
	// 	app.name = Valid(renderApplicationName(app.name, app.script))

	// 	if app.execute_command {
	// 		app.exec_mode = Valid("fork")
	// 	}

	// 	app.username = getCurrentUsername()

	// 	// If command is like pm2 start "python xx.py --ok".
	// 	// Then automatically start the script with bash -c and set a name eq to command
	// 	if app.script.Valid && strings.ContainsRune(app.script.Value, ' ') && !cst.IS_WINDOWS {
	// 		_script := app.script

	// 		if _, err := which.Which("bash"); err == nil {
	// 			app.script = Valid("bash")
	// 			app.args = []string{"-c", _script}
	// 			if !app.name.Valid {
	// 				app.name = _script
	// 			}
	// 		} else if _, err := which.Which("sh"); err == nil {
	// 			app.script = "sh"
	// 			app.args = []string{"-c", _script}
	// 			if !app.name.Valid {
	// 				app.name = _script
	// 			}
	// 		} else {
	// 			warn("bash or sh not available in $PATH, keeping script as is")
	// 		}
	// 	}

	// 	// Add log_date_format by default
	// 	if app.time || process.env.ASZ_MODE {
	// 		app.log_date_format = Valid("2006-01-02T15:04:05")
	// 	}

	// 	// Checks + Resolve UID/GID
	// 	// comes from pm2 --uid <> --gid <> or --user
	// 	if app.uid.Valid || app.gid.Valid || app.user.Valid {
	// 		// 1/ Check if windows
	// 		if cst.IS_WINDOWS {
	// 			printError(cst.PREFIX_MSG_ERR + "--uid and --git does not works on windows")
	// 			return nil, errors.New("--uid and --git does not works on windows")
	// 		}

	// 		// 2/ Verify that user is root (todo: verify if other has right)
	// 		if process.env.NODE_ENV != "test" && process.getuid && process.getuid() != 0 {
	// 			printError(cst.PREFIX_MSG_ERR + "To use --uid and --gid please run pm2 as root")
	// 			return nil, errors.New("To use UID and GID please run PM2 as root")
	// 		}

	// 		// 3/ Resolve user info via /etc/password
	// 		users, err := passwd.GetUsers()
	// 		if err != nil {
	// 			printError(err.Error())
	// 			return nil, err
	// 		}

	// 		user_info, ok := users[app.uid || app.user]
	// 		if !ok {
	// 			printError(`${cst.PREFIX_MSG_ERR} User ${app.uid || app.user} cannot be found`)
	// 			return nil, errors.New(`${cst.PREFIX_MSG_ERR} User ${app.uid || app.user} cannot be found`)
	// 		}

	// 		app.env.HOME = user_info.homedir
	// 		app.uid = user_info.userId

	// 		// 4/ Resolve group id if gid is specified
	// 		if app.gid.Valid {
	// 			groups, err := passwd.getGroups()
	// 			if err != nil {
	// 				printError(err)
	// 				return nil, err
	// 			}
	// 			group_info, ok = groups[app.gid]
	// 			if !ok {
	// 				printError(`${cst.PREFIX_MSG_ERR} Group ${app.gid} cannot be found`)
	// 				return nil, errors.New(`${cst.PREFIX_MSG_ERR} Group ${app.gid} cannot be found`)
	// 			}
	// 			app.gid = group_info.id
	// 		} else {
	// 			app.gid = user_info.groupId
	// 		}
	// 	}

	// 	// Specific options of PM2.io
	// 	app.deep_monitoring = process.env.PM2_DEEP_MONITORING

	// 	if !app.automation {
	// 		app.pmx = false
	// 	}

	// 	if app.disable_trace {
	// 		app.trace = false
	// 	}

	// 	// Instances params
	// 	if app.instances == "max" {
	// 		app.instances = 0
	// 	}

	// 	if typeof(app.instances) == "string" {
	// 		app.instances = parseInt(app.instances) || 0
	// 	}

	// 	if app.exec_mode != "cluster_mode" && !app.instances {
	// 		app.merge_logs = true
	// 	}

	// 	if app.cron_restart {
	// 		var err error
	// 		err = determineCron(app)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	}

	// 	// Now validation configuration
	// 	if err := validateJSON(app); err != nil {
	// 		warn(err)
	// 		return nil, err
	// 	}

	// 	// verifiedConf=append(verifiedConf,ret.config);
	// }

	// return verifiedConf, nil
}

// Get current username
func getCurrentUsername() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil

	// current_user := Or(
	// 	GetEnv("USER"),
	// 	GetEnv("LNAME"),
	// 	GetEnv("USERNAME"),
	// 	GetEnv("SUDO_USER"),
	// 	GetEnv("C9_USER"),
	// 	GetEnv("LOGNAME"),
	// )
}

// Render an app name if not existing.
func renderApplicationName(name Optional[string], script Optional[string]) string {
	if name.Valid {
		return name.Value
	}

	res := OrDefault("undefined", Map(script, path.Base))
	extension := path.Ext(res)
	if extension != "" {
		return strings.TrimRight(res, "."+extension)
	}

	return res
}
