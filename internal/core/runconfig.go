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
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/lo"
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
	StdoutFile fun.Option[string]
	// StderrFile - file to write stderr to
	StderrFile fun.Option[string]
	// Args - arguments for process, not including executable itself as first argument
	Args []string
	// Tags - process tags, excluding `all` tag
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
		Func: func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("wrong number of arguments")
			}

			filename, ok := args[0].(string)
			if !ok {
				return nil, errors.Newf("filename must be a string, but was %q", args[0])
			}

			// TODO: somehow relative to cwd

			data, errRead := os.ReadFile(filename)
			if errRead != nil {
				return nil, errors.Wrapf(errRead, "read env file %q", filename)
			}

			env, errUnmarshal := godotenv.UnmarshalBytes(data)
			if errUnmarshal != nil {
				return nil, errors.Wrapf(errUnmarshal, "parse env file %q", filename)
			}

			return lo.MapValues(env, func(v string, _ string) any {
				return v
			}), nil
		},
		Params: ast.Identifiers{"filename"},
	})
	return vm
}

//nolint:funlen // no
func LoadConfigs(filename string) ([]RunConfig, error) {
	if !isConfigFile(filename) {
		return nil, errors.Newf("invalid config file %q", filename)
	}

	jsonText, err := newVM().EvaluateFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "evaluate jsonnet file")
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
		return nil, errors.Wrapf(err, "unmarshal configs json")
	}

	// validate configs
	errValidation := errors.Combine(fun.Map[error](func(config configScanDTO) error {
		if config.Command == "" {
			return errors.Newf("missing command in config %#v", config)
		}

		return nil
	}, scannedConfigs...)...)
	if errValidation != nil {
		return nil, errValidation
	}

	return fun.MapErr[RunConfig, configScanDTO, error](func(config configScanDTO) (RunConfig, error) {
		watch := fun.Zero[fun.Option[*regexp.Regexp]]()
		if config.Watch != nil {
			re, err := regexp.Compile(*config.Watch)
			if err != nil {
				return fun.Zero[RunConfig](), errors.Wrapf(err, "invalid watch pattern %q", *config.Watch)
			}
			watch = fun.Valid(re)
		}

		relativeCwd := filepath.Join(filepath.Dir(filename), fun.Deref(config.Cwd))
		cwd, err := filepath.Abs(relativeCwd)
		if err != nil {
			return fun.Zero[RunConfig](), errors.Wrapf(err, "get absolute cwd, relative is %q", relativeCwd)
		}

		return RunConfig{
			Name:    fun.FromPtr(config.Name),
			Command: config.Command,
			Args: fun.Map[string](func(arg any, i int) string {
				switch a := arg.(type) {
				case fmt.Stringer:
					return a.String()
				case int, int8, int16, int32, int64,
					uint, uint8, uint16, uint32, uint64,
					float32, float64, bool, string:
					return fmt.Sprint(arg)
				default:
					argStr := fmt.Sprintf("%v", arg)
					log.Error().
						Str("arg", argStr).
						Int("i", i).
						Any("config", config).
						Msg("unknown arg type")

					return argStr
				}
			}, config.Args...),
			Tags: config.Tags,
			Cwd:  cwd,
			Env: lo.MapValues(config.Env, func(name string, value any) string {
				switch v := value.(type) {
				case fmt.Stringer:
					return v.String()
				case int, int8, int16, int32, int64,
					uint, uint8, uint16, uint32, uint64,
					float32, float64, bool, string:
					return fmt.Sprint(v)
				default:
					valStr := fmt.Sprintf("%v", v)
					log.Error().
						Str("value", valStr).
						Str("name", name).
						Any("config", config).
						Msg("unknown env value type")

					return valStr
				}
			}),
			Watch: watch,
			Actions: Actions{
				Healthcheck: nil,
				Custom:      nil,
			},
			StdoutFile:   fun.Zero[fun.Option[string]](),
			StderrFile:   fun.Zero[fun.Option[string]](),
			KillTimeout:  0,
			KillChildren: false,
			Autorestart:  false,
			MaxRestarts:  0,
		}, nil
	}, scannedConfigs...)
}
