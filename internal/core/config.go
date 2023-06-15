package core

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rprtr258/xerr"
)

const Version = "0.0.1"

var (
	ErrConfigNotExists = errors.New("config file not exists")

	_configPath = filepath.Join(DirHome, "config.json")
)

type Config struct {
	Version string
}

var DefaultConfig = Config{
	Version: Version,
}

func ReadConfig() (Config, error) {
	configBytes, errRead := os.ReadFile(_configPath)
	if errRead != nil {
		if errors.Is(errRead, fs.ErrNotExist) {
			return Config{}, ErrConfigNotExists
		}

		return Config{}, xerr.NewWM(errRead, "read config file", xerr.Fields{
			"filename": _configPath,
		})
	}

	var config Config
	if errUnmarshal := json.Unmarshal(configBytes, &config); errUnmarshal != nil {
		return Config{}, xerr.NewWM(errUnmarshal, "parse config")
	}

	return config, nil
}

func WriteConfig(config Config) error {
	configBytes, errMarshal := json.Marshal(config)
	if errMarshal != nil {
		return xerr.NewWM(errMarshal, "marshal config")
	}

	if errWrite := os.WriteFile(_configPath, configBytes, 0o644); errWrite != nil { //nolint:gosec // not unsafe i guess
		return xerr.NewWM(errWrite, "write config", xerr.Fields{
			"filename": _configPath,
		})
	}

	return nil
}
