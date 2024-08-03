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

	"github.com/rprtr258/pm/internal/core/namegen"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/lo"
)

// RunConfig - configuration of process to manage
type RunConfig struct {
	Env         map[string]string          //  environment variables
	Watch       fun.Option[*regexp.Regexp] //  regexp for files to watch and restart on changes
	Command     string                     //  process command, full path
	Cwd         string                     //  working directory
	StdoutFile  fun.Option[string]         //  file to write stdout to
	StderrFile  fun.Option[string]         //  file to write stderr to
	Args        []string                   //  arguments for process, not including executable itself as first argument
	Tags        []string                   //  process tags, excluding `all` tag
	Name        string                     // Name of a process if defined, otherwise generated
	KillTimeout time.Duration              //  before sending SIGKILL after SIGINT
	Autorestart bool                       //  restart process automatically after its death
	MaxRestarts uint                       //  maximum number of restarts, 0 means no limit
	Startup     bool                       //  run process on OS startup
	DependsOn   []string                   // name of processes that must be started before this one
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
				return nil, errors.Newf("wrong number of arguments %d", len(args))
			}

			filename, ok := args[0].(string)
			if !ok {
				return nil, errors.Newf("filename must be a string, but was %T", args[0])
			}

			// TODO: somehow relative to cwd

			data, errRead := os.ReadFile(filename)
			if errRead != nil {
				return nil, errors.Wrapf(errRead, "read file %q", filename)
			}

			env, errUnmarshal := godotenv.UnmarshalBytes(data)
			if errUnmarshal != nil {
				return nil, errors.Wrapf(errUnmarshal, "parse dotenv file %q", filename)
			}

			return env, nil
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
		Name      *string        `json:"name"`
		Cwd       *string        `json:"cwd"`
		Env       map[string]any `json:"env"`
		Command   string         `json:"command"`
		Args      []any          `json:"args"`
		Tags      []string       `json:"tags"`
		Watch     *string        `json:"watch"`
		Startup   bool           `json:"startup"`
		DependsOn []string       `json:"depends_on"`
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

	return fun.MapErr[RunConfig](func(config configScanDTO) (RunConfig, error) {
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
			Name:    fun.FromPtr(config.Name).OrDefault(namegen.New()),
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
			Watch:       watch,
			StdoutFile:  fun.Zero[fun.Option[string]](),
			StderrFile:  fun.Zero[fun.Option[string]](),
			KillTimeout: 0,
			Autorestart: false,
			MaxRestarts: 0,
			Startup:     config.Startup,
			DependsOn:   config.DependsOn,
		}, nil
	}, scannedConfigs...)
}
