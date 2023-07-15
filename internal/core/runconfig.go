package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/joho/godotenv"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

type Actions struct {
	Healthcheck any // TODO: tcp port listen check/http check/command run
	// Custom actions
	Custom map[string]struct {
		Command string
		Args    []string
	}
}

// RunConfig - configuration of process to manage
type RunConfig struct {
	// Env - environment variables
	Env map[string]string
	// Actions to perform on process by name
	Actions Actions
	// Watch - regexp for files to watch and restart on changes
	Watch fun.Option[*regexp.Regexp]
	// Command - process command, full path
	Command string
	// Cwd - working directory
	Cwd string
	// StdoutFile - file to write stdout to
	StdoutFile string
	// StderrFile - file to write stderr to
	StderrFile string
	// Args - arguments for process, not including executable itself as first argument
	Args []string
	// Tags - process tags, exluding `all` tag
	Tags []string
	// Name of a process if defined, otherwise generated
	Name fun.Option[string]
	// KillTimeout - before sending SIGKILL after SIGINT
	// TODO: use
	KillTimeout time.Duration
	// KillChildren - stop children processes on process stop
	// TODO: use
	KillChildren bool
	// Autorestart - restart process automatically after its death
	Autorestart bool
	// MaxRestarts - maximum number of restarts, 0 means no limit
	MaxRestarts uint
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
	vm.NativeFunction(&jsonnet.NativeFunction{
		Name: "dotenv",
		Func: func(args []interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("wrong number of arguments")
			}

			filename, ok := args[0].(string)
			if !ok {
				return nil, xerr.NewM("filename must be a string", xerr.Fields{"filename": args[0]})
			}

			data, errRead := os.ReadFile(filename)
			if errRead != nil {
				return nil, xerr.NewWM(errRead, "read env file", xerr.Fields{"filename": filename})
			}

			env, errUnmarshal := godotenv.UnmarshalBytes(data)
			if errUnmarshal != nil {
				return nil, xerr.NewWM(errUnmarshal, "parse env file", xerr.Fields{"filename": filename})
			}

			return lo.MapValues(env, func(v string, _ string) any {
				return v
			}), nil
		},
		Params: ast.Identifiers{"filename"},
	})
	return vm
}

func LoadConfigs(filename string) ([]RunConfig, error) {
	if !isConfigFile(filename) {
		return nil, xerr.NewM(
			"invalid config file",
			xerr.Fields{"configFilename": filename},
			xerr.Stacktrace,
		)
	}

	jsonText, err := newVM().EvaluateFile(filename)
	if err != nil {
		return nil, xerr.NewWM(err, "evaluate jsonnet file")
	}

	type configScanDTO struct {
		Name    *string        `json:"name"`
		Cwd     *string        `json:"cwd"`
		Env     map[string]any `json:"env"`
		Command string         `json:"command"`
		Args    []any          `json:"args"`
		Tags    []string       `json:"tags"`
		Watch   *string        `json:"watch"`
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
		watch := fun.Option[*regexp.Regexp]{}
		if config.Watch != nil {
			re, err := regexp.Compile(*config.Watch)
			if err != nil {
				// TODO: fail
				slog.Error("invalid watch pattern", "pattern", *config.Watch)
			}
			watch = fun.Valid(re)
		}

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
					slog.Error("unknown arg type",
						"arg", argStr,
						"i", i,
						"config", config,
					)

					return argStr
				}
			}),
			Tags: config.Tags,
			Cwd: lo.
				If(config.Cwd == nil, filepath.Dir(filename)).
				ElseF(func() string { return *config.Cwd }),
			Env: lo.MapValues(config.Env, func(value any, name string) string {
				switch v := value.(type) {
				case fmt.Stringer:
					return v.String()
				case int, int8, int16, int32, int64,
					uint, uint8, uint16, uint32, uint64,
					float32, float64, bool, string:
					return fmt.Sprint(v)
				default:
					valStr := fmt.Sprintf("%v", v)
					slog.Error("unknown env value type",
						"value", valStr,
						"name", name,
						"config", config,
					)

					return valStr
				}
			}),
			Watch: watch,
		}
	}), nil
}
