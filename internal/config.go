package internal

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/multierr"

	"github.com/rprtr258/pm/internal/logging"
)

type config struct {
	// IsWindows              bool
	// LowMemoryEnvironment   bool
	// MachineName            Optional[string]
	// SecretKey              Optional[string]
	// PublicKey              Optional[string]
	// KeymetricsRootURL      string
	// ExpBackoffResetTimer   time.Duration
	// RemotePortTCP          int
	// ReloadLockTimeout      time.Duration
	// GracefuleTimeout       time.Duration
	// GracefuleListenTimeout time.Duration
	// AggregationDuration    time.Duration
	// TraceFlushInterval     time.Duration
	// // ConcurrentActions when doing start/restart/reload
	// ConcurrentActions int
	// Debug             bool
	// WebIpAddr         string
	// WebPort           int
	// WebStripEnvVars   bool
	// ModifyRequire     bool
	// WorkerInterval    time.Duration
	// KillTimeout       time.Duration
	// KillSignal        os.Signal
	// KillUseMessage    bool
	Programmatic bool
	// LogDateFormat     string
	Silent bool
}

func (cfg config) L() logging.Logger {
	return logging.Logger(cfg.Silent || cfg.Programmatic)
}

func readConfig() (config, error) {
	// pm2Debug := envIsDefined("PM2_DEBUG")

	var merr error
	cfg := config{
		Silent: envIsDefined("PM2_SILENT"),
		// 	IsWindows:            goos.IsWindows != 0, //process.platform == "win32" || process.platform == "win64" || process.env.OSTYPE == "msys" || process.env.OSTYPE == "cygwin",
		// 	LowMemoryEnvironment: false,               // process.env.PM2_OPTIMIZE_MEMORY
		// 	MachineName: Or(
		// 		GetEnv("INSTANCE_NAME"),
		// 		GetEnv("MACHINE_NAME"),
		// 		GetEnv("PM2_MACHINE_NAME"),
		// 	),
		// 	SecretKey: Or(
		// 		GetEnv("KEYMETRICS_SECRET"),
		// 		GetEnv("PM2_SECRET_KEY"),
		// 		GetEnv("SECRET_KEY"),
		// 	),
		// 	PublicKey: Or(
		// 		GetEnv("KEYMETRICS_PUBLIC"),
		// 		GetEnv("PM2_PUBLIC_KEY"),
		// 		GetEnv("PUBLIC_KEY"),
		// 	),
		// 	KeymetricsRootURL: OrDefault(
		// 		"root.keymetrics.io",
		// 		GetEnv("KEYMETRICS_NODE"),
		// 		GetEnv("PM2_APM_ADDRESS"),
		// 		GetEnv("ROOT_URL"),
		// 		GetEnv("INFO_NODE"),
		// 	),
		// 	ExpBackoffResetTimer:   envDurationOrDefault(&merr, "EXP_BACKOFF_RESET_TIMER", 30000*time.Millisecond),
		// 	RemotePortTCP:          envIntOrDefault(&merr, "KEYMETRICS_PUSH_PORT", 80),
		// 	ReloadLockTimeout:      envDurationOrDefault(&merr, "PM2_RELOAD_LOCK_TIMEOUT", 30000*time.Millisecond),
		// 	GracefuleTimeout:       envDurationOrDefault(&merr, "PM2_GRACEFUL_TIMEOUT", 8000*time.Millisecond),
		// 	GracefuleListenTimeout: envDurationOrDefault(&merr, "PM2_GRACEFUL_LISTEN_TIMEOUT", 3000*time.Millisecond),
		// 	AggregationDuration:    _if(pm2Debug, 3000*time.Millisecond, 5*60000*time.Millisecond),
		// 	TraceFlushInterval:     _if(pm2Debug, 1000*time.Millisecond, 60000*time.Millisecond),

		// 	ConcurrentActions: envIntOrDefault(&merr, "PM2_CONCURRENT_ACTIONS", 2),

		// 	Debug:           pm2Debug,
		// 	WebIpAddr:       envStringOrDefault(&merr, "PM2_API_IPADDR", "0.0.0.0"),
		// 	WebPort:         envIntOrDefault(&merr, "PM2_API_PORT", 9615),
		// 	WebStripEnvVars: envIsDefined("PM2_WEB_STRIP_ENV_VARS"),
		// 	ModifyRequire:   envIsDefined("PM2_MODIFY_REQUIRE"),

		// 	WorkerInterval: envDurationOrDefault(&merr, "PM2_WORKER_INTERVAL", 30000*time.Millisecond),
		// 	KillTimeout:    envDurationOrDefault(&merr, "PM2_KILL_TIMEOUT", 1600*time.Millisecond),
		// 	KillSignal:     os.Interrupt, // PM2_KILL_SIGNAL || "SIGINT"
		// 	KillUseMessage: envIsDefined("PM2_KILL_USE_MESSAGE"),

		Programmatic: envIsDefined("PM2_PROGRAMMATIC"),
		// 	LogDateFormat: envStringOrDefault(&merr, "PM2_LOG_DATE_FORMAT", "2006-01-02T15:04:05"),

		// 	// path_structure = require("./paths.js")(process.env.OVER_HOME);
	}
	return cfg, merr
}

func envIsDefined(varName string) bool {
	_, ok := os.LookupEnv(varName)
	return ok
}

func envIntOrDefault(merr *error, varName string, defaultValue int) int {
	varValue, ok := os.LookupEnv(varName)
	if !ok {
		return defaultValue
	}

	res, err := strconv.Atoi(varValue)
	multierr.AppendInto(merr, fmt.Errorf("failed reading int from %s: %w", varName, err))
	return res
}

func envStringOrDefault(merr *error, varName string, defaultValue string) string {
	varValue, ok := os.LookupEnv(varName)
	if !ok {
		return defaultValue
	}

	return varValue
}

func envDurationOrDefault(merr *error, varName string, defaultValue time.Duration) time.Duration {
	varValue, ok := os.LookupEnv(varName)
	if !ok {
		return defaultValue
	}

	res, err := time.ParseDuration(varValue)
	multierr.AppendInto(merr, fmt.Errorf("failed reading duration from %s: %w", varName, err))
	return res
}

func _if[T any](predicate bool, ifTrue, ifFalse T) T {
	if predicate {
		return ifTrue
	}
	return ifFalse
}

// var util    = require('util');

// /**
//  * Validator of configured file / commander options.
//  */
// var Config = module.exports = {
//   _errMsgs: {
//     'require': '"%s" is required',
//     'type'   : 'Expect "%s" to be a typeof %s, but now is %s',
//     'regex'  : 'Verify "%s" with regex failed, %s',
//     'max'    : 'The maximum of "%s" is %s, but now is %s',
//     'min'    : 'The minimum of "%s" is %s, but now is %s'
//   },
//   /**
//    * Schema definition.
//    * @returns {exports|*}
//    */
//   get schema(){
//     // Cache.
//     if (this._schema) {
//       return this._schema;
//     }
//     // Render aliases.
//     this._schema = require('../API/schema');
//     for (var k in this._schema) {
//       if (k.indexOf('\\') > 0) {
//         continue;
//       }
//       var aliases = [
//         k.split('_').map(function(n, i){
//           if (i != 0 && n && n.length > 1) {
//             return n[0].toUpperCase() + n.slice(1);
//           }
//           return n;
//         }).join('')
//       ];

//       if (this._schema[k].alias && Array.isArray(this._schema[k].alias)) {
//         // If multiple aliases, merge
//         this._schema[k].alias.forEach(function(alias) {
//           aliases.splice(0, 0, alias);
//         });
//       }
//       else if (this._schema[k].alias)
//         aliases.splice(0, 0, this._schema[k].alias);

//       this._schema[k].alias = aliases;
//     }
//     return this._schema;
//   }
// };

// Verify JSON configurations.
func validateJSON(json any) error {
	// TODO: validate config
	return nil
}

// /**
//  * Validate key-value pairs by specific schema
//  * @param {String} key
//  * @param {Mixed} value
//  * @param {Object} sch
//  * @returns {*}
//  * @private
//  */
// Config._valid = function(key, value, sch){
//   var sch = sch || this.schema[key],
//       scht = typeof sch.type == 'string' ? [sch.type] : sch.type;

//   // Required value.
//   var undef = typeof value == 'undefined';
//   if(this._error(sch.require && undef, 'require', key)){
//     return null;
//   }

//   // If undefined, make a break.
//   if (undef) {
//     return null;
//   }

//   // Wrap schema types.
//   scht = scht.map(function(t){
//     return '[object ' + t[0].toUpperCase() + t.slice(1) + ']'
//   });

//   // Typeof value.
//   var type = Object.prototype.toString.call(value), nt = '[object Number]';

//   // Auto parse Number
//   if (type != '[object Boolean]' && scht.indexOf(nt) >= 0 && !isNaN(value)) {
//     value = parseFloat(value);
//     type = nt;
//   }

//   // Verify types.
//   if (this._error(!~scht.indexOf(type), 'type', key, scht.join(' / '), type)) {
//     return null;
//   }

//   // Verify RegExp if exists.
//   if (this._error(type == '[object String]' && sch.regex && !(new RegExp(sch.regex)).test(value),
//       'regex', key, sch.desc || ('should match ' + sch.regex))) {
//     return null;
//   }

//   // Verify maximum / minimum of Number value.
//   if (type == '[object Number]') {
//     if (this._error(typeof sch.max != 'undefined' && value > sch.max, 'max', key, sch.max, value)) {
//       return null;
//     }
//     if (this._error(typeof sch.min != 'undefined' && value < sch.min, 'min', key, sch.min, value)) {
//       return null;
//     }
//   }

//   // If first type is Array, but current is String, try to split them.
//   if(scht.length > 1 && type != scht[0] && type == '[object String]'){
//     if(scht[0] == '[object Array]') {
//       // unfortunately, js does not support lookahead RegExp (/(?<!\\)\s+/) now (until next ver).
//       value = value.split(/([\w\-]+\="[^"]*")|([\w\-]+\='[^']*')|"([^"]*)"|'([^']*)'|\s/)
//         .filter(function(v){
//           return v && v.trim();
//         });
//     }
//   }

//   // Custom types: sbyte && stime.
//   if(sch.ext_type && type == '[object String]' && value.length >= 2) {
//     var seed = {
//       'sbyte': {
//         'G': 1024 * 1024 * 1024,
//         'M': 1024 * 1024,
//         'K': 1024
//       },
//       'stime': {
//         'h': 60 * 60 * 1000,
//         'm': 60 * 1000,
//         's': 1000
//       }
//     }[sch.ext_type];

//     if(seed){
//       value = parseFloat(value.slice(0, -1)) * (seed[value.slice(-1)]);
//     }
//   }
//   return value;
// };

// /**
//  * Wrap errors.
//  * @param {Boolean} possible A value indicates whether it is an error or not.
//  * @param {String} type
//  * @returns {*}
//  * @private
//  */
// Config._error = function(possible, type){
//   if (possible) {
//     var args = Array.prototype.slice.call(arguments);
//     args.splice(0, 2, this._errMsgs[type]);
//     this._errors && this._errors.push(util.format.apply(null, args));
//   }
//   return possible;
// }
