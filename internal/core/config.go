package core

import (
	"encoding/json"
	stdErrors "errors"
	"io/fs"
	"os"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/errors"
)

// NOTE: being set at compile time using ldflags
var Version = "dev"

type Config struct {
	Version        string
	Debug          bool
	DirLogs, DirDB string
}

var DefaultConfig = Config{
	Version: Version,
	Debug:   false,
	DirLogs: _dirProcsLogs,
	DirDB:   _dirDB,
}

func writeConfig(config Config) error {
	configBytes, errMarshal := json.Marshal(config)
	if errMarshal != nil {
		return errors.Wrapf(errMarshal, "marshal config")
	}

	if errWrite := os.WriteFile(_configPath, configBytes, 0o640); errWrite != nil {
		return errors.Wrapf(errWrite, "write config %q", _configPath)
	}

	return nil
}

func ReadConfig() (Config, error) {
	configBytes, errRead := os.ReadFile(_configPath)
	if errRead != nil {
		if stdErrors.Is(errRead, fs.ErrNotExist) {
			log.Info().Str("filename", _configPath).Msg("writing initial config...")

			if errWrite := writeConfig(DefaultConfig); errWrite != nil {
				return fun.Zero[Config](), errors.Wrapf(errWrite, "write initial config")
			}

			return DefaultConfig, nil
		}

		return fun.Zero[Config](), errors.Wrapf(errRead, "read config file %q", _configPath)
	}

	var config Config
	if errUnmarshal := json.Unmarshal(configBytes, &config); errUnmarshal != nil {
		return fun.Zero[Config](), errors.Wrapf(errUnmarshal, "parse config")
	}
	config.DirLogs = _dirProcsLogs
	config.DirDB = _dirDB
	return config, nil
}
