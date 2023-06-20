package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-jsonnet"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
)

type RunConfig struct {
	Args    []string
	Tags    []string
	Command string
	Cwd     string
	Name    fun.Option[string]
}

func isConfigFile(arg string) bool {
	stat, err := os.Stat(arg)
	if err != nil {
		return false
	}

	return !stat.IsDir()
}

func newVM() *jsonnet.VM {
	vm := jsonnet.MakeVM()
	vm.ExtVar("now", time.Now().Format("15:04:05"))
	return vm
}

func LoadConfigs(filename string) ([]RunConfig, error) {
	if !isConfigFile(filename) {
		return nil, xerr.NewM(
			"invalid config file",
			xerr.Fields{"configFilename": filename},
			xerr.Stacktrace(0),
		)
	}

	jsonText, err := newVM().EvaluateFile(filename)
	if err != nil {
		return nil, xerr.NewWM(err, "evaluate jsonnet file")
	}

	type configScanDTO struct {
		Name    *string  `json:"name"`
		Cwd     *string  `json:"cwd"`
		Command string   `json:"command"`
		Args    []any    `json:"args"`
		Tags    []string `json:"tags"`
	}
	var scannedConfigs []configScanDTO
	if err := json.Unmarshal([]byte(jsonText), &scannedConfigs); err != nil {
		return nil, xerr.NewWM(err, "unmarshal configs json")
	}

	// validate configs
	errValidation := xerr.Combine(lo.Map(scannedConfigs, func(config configScanDTO, _ int) error {
		if config.Command == "" {
			return xerr.NewM(
				"missing command",
				xerr.Fields{"config": config},
			)
		}

		return nil
	})...)
	if errValidation != nil {
		return nil, errValidation
	}

	return lo.Map(scannedConfigs, func(config configScanDTO, _ int) RunConfig {
		return RunConfig{
			Name:    fun.FromPtr(config.Name),
			Command: config.Command,
			Args: lo.Map(config.Args, func(arg any, i int) string {
				switch a := arg.(type) {
				case fmt.Stringer:
					return a.String()
				case int, int8, int16, int32, int64,
					uint, uint8, uint16, uint32, uint64,
					float32, float64, bool, string:
					return fmt.Sprint(arg)
				default:
					argStr := fmt.Sprintf("%v", arg)
					log.Errorf("unknown arg type", log.F{
						"arg":    argStr,
						"i":      i,
						"config": config,
					})

					return argStr
				}
			}),
			Tags: config.Tags,
			Cwd: lo.
				If(config.Cwd == nil, filepath.Dir(filename)).
				ElseF(func() string { return *config.Cwd }),
		}
	}), nil
}
