package constants

import (
	"github.com/fatih/color"
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
)

const (
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
