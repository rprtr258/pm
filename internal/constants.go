package internal

import (
	"fmt"
	"internal/goos"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	"go.uber.org/multierr"
	// // debug  "debug/_call_/pm2/_colon_/conf"
)

var (
	cyanColor = color.New(color.FgCyan)
	redColor  = color.New(color.FgRed)

	PREFIX_MSG         = color.GreenString("[PM2] ")
	PREFIX_MSG_INFO    = cyanColor.Sprint("[PM2][INFO] ")
	PREFIX_MSG_ERR     = redColor.Sprint("[PM2][ERROR] ")
	PREFIX_MSG_MOD     = color.New(color.FgGreen, color.Bold).Sprint("[PM2][Module] ")
	PREFIX_MSG_MOD_ERR = redColor.Sprint("[PM2][Module][ERROR] ")
	PREFIX_MSG_WARNING = color.New(color.FgYellow).Sprint("[PM2][WARN] ")
	PREFIX_MSG_SUCCESS = cyanColor.Sprint("[PM2] ")

	PM2_IO_MSG     = cyanColor.Sprint("[PM2 I/O]")
	PM2_IO_MSG_ERR = redColor.Sprint("[PM2 I/O]")

	// TODO: include in binary
	TEMPLATE_FOLDER = "assets/templates"

	APP_CONF_DEFAULT_FILE = "ecosystem.config.js"
	APP_CONF_TPL          = "ecosystem.tpl"
	APP_CONF_TPL_SIMPLE   = "ecosystem-simple.tpl"
	SAMPLE_CONF_FILE      = "sample-conf.js"
	LOGROTATE_SCRIPT      = "logrotate.d/pm2"

	DOCKERFILE_NODEJS = "Dockerfiles/Dockerfile-nodejs.tpl"
	DOCKERFILE_JAVA   = "Dockerfiles/Dockerfile-java.tpl"
	DOCKERFILE_RUBY   = "Dockerfiles/Dockerfile-ruby.tpl"

	SUCCESS_EXIT           = 0
	ERROR_EXIT             = 1
	CODE_UNCAUGHTEXCEPTION = 2

	ONLINE_STATUS     = "online"
	STOPPED_STATUS    = "stopped"
	STOPPING_STATUS   = "stopping"
	WAITING_RESTART   = "waiting restart"
	LAUNCHING_STATUS  = "launching"
	ERRORED_STATUS    = "errored"
	ONE_LAUNCH_STATUS = "one-launch-status"

	CLUSTER_MODE_ID = "cluster_mode"
	FORK_MODE_ID    = "fork_mode"

	PM2_BANNER          = "../lib/motd"
	PM2_UPDATE          = "../lib/API/pm2-plus/pres/motd.update"
	DEFAULT_MODULE_JSON = "package.json"

	MODULE_BASEFOLDER      = "module"
	MODULE_CONF_PREFIX     = "module-db-v2"
	MODULE_CONF_PREFIX_TAR = "tar-modules"

	REMOTE_PORT      = 41624
	REMOTE_HOST      = "s1.keymetrics.io"
	SEND_INTERVAL    = 1000
	LOGS_BUFFER_SIZE = 8
	CONTEXT_ON_ERROR = 2
)

type config struct {
	IsWindows              bool
	LowMemoryEnvironment   bool
	MachineName            Optional[string]
	SecretKey              Optional[string]
	PublicKey              Optional[string]
	KeymetricsRootURL      string
	ExpBackoffResetTimer   time.Duration
	RemotePortTCP          int
	ReloadLockTimeout      time.Duration
	GracefuleTimeout       time.Duration
	GracefuleListenTimeout time.Duration
	AggregationDuration    time.Duration
	TraceFlushInterval     time.Duration
	// ConcurrentActions when doing start/restart/reload
	ConcurrentActions int
	Debug             bool
	WebIpAddr         string
	WebPort           int
	WebStripEnvVars   bool
	ModifyRequire     bool
	WorkerInterval    time.Duration
	KillTimeout       time.Duration
	KillSignal        os.Signal
	KillUseMessage    bool
	Programmatic      bool
	LogDateFormat     string
}

func readConfig() (config, error) {
	pm2Debug := envIsDefined("PM2_DEBUG")

	var merr error
	cfg := config{
		IsWindows:            goos.IsWindows != 0, //process.platform == "win32" || process.platform == "win64" || process.env.OSTYPE == "msys" || process.env.OSTYPE == "cygwin",
		LowMemoryEnvironment: false,               // process.env.PM2_OPTIMIZE_MEMORY
		MachineName: Or(
			GetEnv("INSTANCE_NAME"),
			GetEnv("MACHINE_NAME"),
			GetEnv("PM2_MACHINE_NAME"),
		),
		SecretKey: Or(
			GetEnv("KEYMETRICS_SECRET"),
			GetEnv("PM2_SECRET_KEY"),
			GetEnv("SECRET_KEY"),
		),
		PublicKey: Or(
			GetEnv("KEYMETRICS_PUBLIC"),
			GetEnv("PM2_PUBLIC_KEY"),
			GetEnv("PUBLIC_KEY"),
		),
		KeymetricsRootURL: OrDefault(
			"root.keymetrics.io",
			GetEnv("KEYMETRICS_NODE"),
			GetEnv("PM2_APM_ADDRESS"),
			GetEnv("ROOT_URL"),
			GetEnv("INFO_NODE"),
		),
		ExpBackoffResetTimer:   envDurationOrDefault(&merr, "EXP_BACKOFF_RESET_TIMER", 30000*time.Millisecond),
		RemotePortTCP:          envIntOrDefault(&merr, "KEYMETRICS_PUSH_PORT", 80),
		ReloadLockTimeout:      envDurationOrDefault(&merr, "PM2_RELOAD_LOCK_TIMEOUT", 30000*time.Millisecond),
		GracefuleTimeout:       envDurationOrDefault(&merr, "PM2_GRACEFUL_TIMEOUT", 8000*time.Millisecond),
		GracefuleListenTimeout: envDurationOrDefault(&merr, "PM2_GRACEFUL_LISTEN_TIMEOUT", 3000*time.Millisecond),
		AggregationDuration:    _if(pm2Debug, 3000*time.Millisecond, 5*60000*time.Millisecond),
		TraceFlushInterval:     _if(pm2Debug, 1000*time.Millisecond, 60000*time.Millisecond),

		ConcurrentActions: envIntOrDefault(&merr, "PM2_CONCURRENT_ACTIONS", 2),

		Debug:           pm2Debug,
		WebIpAddr:       envStringOrDefault(&merr, "PM2_API_IPADDR", "0.0.0.0"),
		WebPort:         envIntOrDefault(&merr, "PM2_API_PORT", 9615),
		WebStripEnvVars: envIsDefined("PM2_WEB_STRIP_ENV_VARS"),
		ModifyRequire:   envIsDefined("PM2_MODIFY_REQUIRE"),

		WorkerInterval: envDurationOrDefault(&merr, "PM2_WORKER_INTERVAL", 30000*time.Millisecond),
		KillTimeout:    envDurationOrDefault(&merr, "PM2_KILL_TIMEOUT", 1600*time.Millisecond),
		KillSignal:     os.Interrupt, // PM2_KILL_SIGNAL || "SIGINT"
		KillUseMessage: envIsDefined("PM2_KILL_USE_MESSAGE"),

		Programmatic:  envIsDefined("PM2_PROGRAMMATIC"),
		LogDateFormat: envStringOrDefault(&merr, "PM2_LOG_DATE_FORMAT", "2006-01-02T15:04:05"),

		// path_structure = require("./paths.js")(process.env.OVER_HOME);
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
