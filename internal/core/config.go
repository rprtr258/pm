package core

import (
	"encoding/json"
	stdErrors "errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/pm/internal/infra/errors"
)

// TODO: set at compile time
// see https://developers.redhat.com/articles/2022/11/14/3-ways-embed-commit-hash-go-programs#2__using_go_generate
const Version = "0.1.0"

var (
	ErrConfigNotExists = errors.New("config file not exists")

	_configPath = filepath.Join(DirHome, "config.json")
)

type Config struct {
	Version string
	Debug   bool
}

var DefaultConfig = Config{
	Version: Version,
	Debug:   false,
}

func ReadConfig() (Config, string, error) {
	configBytes, errRead := os.ReadFile(_configPath)
	if errRead != nil {
		if stdErrors.Is(errRead, fs.ErrNotExist) {
			return Config{}, "", ErrConfigNotExists
		}

		return fun.Zero[Config](), "", errors.Wrapf(errRead, "read config file %q", _configPath)
	}

	var config Config
	if errUnmarshal := json.Unmarshal(configBytes, &config); errUnmarshal != nil {
		return fun.Zero[Config](), "", errors.Wrapf(errUnmarshal, "parse config")
	}

	return config, _configPath, nil
}

func WriteConfig(config Config) error {
	configBytes, errMarshal := json.Marshal(config)
	if errMarshal != nil {
		return errors.Wrapf(errMarshal, "marshal config")
	}

	if errWrite := os.WriteFile(_configPath, configBytes, 0o644); errWrite != nil { //nolint:gosec // not unsafe i guess
		return errors.Wrapf(errWrite, "write config %q", _configPath)
	}

	return nil
}
