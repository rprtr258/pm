package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/joho/godotenv"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/rprtr258/fun"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"

	"github.com/rprtr258/pm/internal/errors"
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
	Cron        fun.Option[string]         // cron expression
}

type configScanDTO struct {
	Name      *string           `json:"name"`
	Cwd       *string           `json:"cwd"`
	Env       map[string]string `json:"env"`
	Command   string            `json:"command"`
	Args      []any             `json:"args"`
	Tags      []string          `json:"tags"`
	Watch     *string           `json:"watch"`
	Startup   bool              `json:"startup"`
	DependsOn []string          `json:"depends_on" yaml:"depends_on"`
	Cron      *string           `json:"cron"`
}

type configNamedKeyDTO struct {
	Cwd       *string           `json:"cwd"`
	Env       map[string]string `json:"env"`
	Command   string            `json:"command"`
	Args      []any             `json:"args"`
	Tags      []string          `json:"tags"`
	Watch     *string           `json:"watch"`
	Startup   bool              `json:"startup"`
	DependsOn []string          `json:"depends_on" yaml:"depends_on"`
	Cron      *string           `json:"cron"`
}

type hclProcess struct {
	Name      string            `hcl:"name,label"`
	Cwd       *string           `hcl:"cwd,optional"`
	Env       map[string]string `hcl:"env,optional"`
	Command   string            `hcl:"command"`
	Args      []string          `hcl:"args,optional"`
	Tags      []string          `hcl:"tags,optional"`
	Watch     *string           `hcl:"watch,optional"`
	Startup   bool              `hcl:"startup,optional"`
	DependsOn []string          `hcl:"depends_on,optional"`
	Cron      *string           `hcl:"cron,optional"`
	Body      hcl.Body          `hcl:",remain"`
}

type hclConfigFile struct {
	Processes []*hclProcess `hcl:"process,block"`
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

			data, ok := args[0].(string)
			if !ok {
				return nil, errors.Newf("dotenv must be a string, but was %T", args[0])
			}

			env, errUnmarshal := godotenv.Unmarshal(data)
			if errUnmarshal != nil {
				return nil, errors.Wrapf(errUnmarshal, "parse dotenv")
			}

			// since "evaluate jsonnet file: RUNTIME ERROR: Not a json type: map[string]string"
			res := make(map[string]any, len(env))
			for k, v := range env {
				res[k] = v
			}
			return res, nil
		},
		Params: ast.Identifiers{"dotenv"},
	})
	return vm
}

func LoadConfigs(filename string) ([]RunConfig, error) {
	configs, err := loadConfigs(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "load config file %q", filename)
	}

	return parseConfigsFromDTO(configs, filepath.Dir(filename))
}

func loadConfigs(filename string) ([]configScanDTO, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "stat")
	} else if stat.IsDir() {
		return nil, errors.Newf("config file is actually a directory")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "read config file")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jsonnet":
		jsonText, err := newVM().EvaluateFile(filename)
		if err != nil {
			return nil, errors.Wrapf(err, "evaluate jsonnet file")
		}
		return loadJSONConfigs([]byte(jsonText))
	case ".yaml", ".yml":
		return loadYAMLConfigs(data)
	case ".toml":
		return loadTOMLConfigs(data)
	case ".ini", ".cfg", ".conf":
		return loadINIConfigs(data)
	case ".hcl":
		return loadHCLConfigs(data, filename)
	case ".json":
		return loadJSONConfigs(data)
	default:
		return nil, errors.Newf("unsupported config format %q", ext)
	}
}

func loadYAMLConfigs(data []byte) ([]configScanDTO, error) {
	var configs map[string]configNamedKeyDTO
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return nil, errors.Wrap(err, "parse yaml")
	}

	arrayConfigs := make([]configScanDTO, 0, len(configs))
	for name, config := range configs {
		arrayConfigs = append(arrayConfigs, configScanDTO{
			Name:      fun.Ptr(name),
			Cwd:       config.Cwd,
			Env:       config.Env,
			Command:   config.Command,
			Args:      config.Args,
			Tags:      config.Tags,
			Watch:     config.Watch,
			Startup:   config.Startup,
			DependsOn: config.DependsOn,
			Cron:      config.Cron,
		})
	}
	return arrayConfigs, nil
}

func loadTOMLConfigs(data []byte) ([]configScanDTO, error) {
	var configs map[string]configNamedKeyDTO
	if err := toml.Unmarshal(data, &configs); err != nil {
		return nil, errors.Wrapf(err, "parse toml")
	}

	return fun.MapToSlice(configs, func(name string, cfg configNamedKeyDTO) configScanDTO {
		return configScanDTO{
			Name:      fun.Ptr(name),
			Cwd:       cfg.Cwd,
			Env:       cfg.Env,
			Command:   cfg.Command,
			Args:      cfg.Args,
			Tags:      cfg.Tags,
			Watch:     cfg.Watch,
			Startup:   cfg.Startup,
			DependsOn: cfg.DependsOn,
			Cron:      cfg.Cron,
		}
	}), nil
}

func loadINIConfigs(data []byte) ([]configScanDTO, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, errors.Wrapf(err, "parse ini file")
	}

	configs := make([]configScanDTO, 0, len(cfg.Sections()))
	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}

		sectionName := section.Name()
		configs = append(configs, configScanDTO{
			Name:      &sectionName,
			Cwd:       getStringPtr(section, "cwd"),
			Env:       getEnvMap(section),
			Command:   section.Key("command").String(),
			Args:      getArgsSlice(section),
			Tags:      getStringSlice(section, "tags"),
			Watch:     getStringPtr(section, "watch"),
			Startup:   section.Key("startup").MustBool(false),
			DependsOn: getStringSlice(section, "depends_on"),
			Cron:      getStringPtr(section, "cron"),
		})
	}

	return configs, nil
}

func loadHCLConfigs(data []byte, filename string) ([]configScanDTO, error) {
	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{},
		Functions: map[string]function.Function{
			"now": function.New(&function.Spec{
				Description:  "Get current time",
				Params:       []function.Parameter{},
				Type:         function.StaticReturnType(cty.String),
				VarParam:     nil,
				RefineResult: nil,
				Impl: func([]cty.Value, cty.Type) (cty.Value, error) {
					return cty.StringVal(time.Now().Format("15:04:05")), nil
				},
			}),
			"dotenv": function.New(&function.Spec{
				Description: "Load dotenv file",
				Params: []function.Parameter{
					{
						Name:             "dotenv",
						Description:      "Dotenv file name",
						Type:             cty.String,
						AllowNull:        false,
						AllowUnknown:     false,
						AllowDynamicType: false,
						AllowMarked:      false,
					},
				},
				Type:         function.StaticReturnType(cty.Map(cty.String)),
				VarParam:     nil,
				RefineResult: nil,
				Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
					if len(args) != 1 {
						return cty.Value{}, errors.Newf("wrong number of arguments %d", len(args))
					}

					data := args[0].AsString()

					env, errUnmarshal := godotenv.Unmarshal(data)
					if errUnmarshal != nil {
						return cty.Value{}, errors.Wrapf(errUnmarshal, "parse dotenv")
					}

					// since "evaluate jsonnet file: RUNTIME ERROR: Not a json type: map[string]string"
					res := make(map[string]cty.Value, len(env))
					for k, v := range env {
						res[k] = cty.StringVal(v)
					}
					return cty.MapVal(res), nil
				},
			}),
		},
	}

	var config hclConfigFile
	if err := hclsimple.Decode(filename, data, ctx, &config); err != nil {
		return nil, errors.Wrapf(err, "parse hcl file")
	}

	return fun.Map[configScanDTO](func(proc *hclProcess) configScanDTO {
		return configScanDTO{
			Name:      &proc.Name,
			Cwd:       proc.Cwd,
			Env:       proc.Env,
			Command:   proc.Command,
			Args:      fun.Map[any](func(arg string) any { return arg }, proc.Args...),
			Tags:      proc.Tags,
			Watch:     proc.Watch,
			Startup:   proc.Startup,
			DependsOn: proc.DependsOn,
			Cron:      proc.Cron,
		}
	}, config.Processes...), nil
}

func loadJSONConfigs(data []byte) ([]configScanDTO, error) {
	var scannedConfigs []configScanDTO
	if err := json.Unmarshal(data, &scannedConfigs); err != nil {
		return nil, errors.Wrapf(err, "unmarshal configs json")
	}

	return scannedConfigs, nil
}

func parseConfigsFromDTO(configs []configScanDTO, dir string) ([]RunConfig, error) {
	// validate configs
	errValidation := errors.Combine(fun.Map[error](func(config configScanDTO) error {
		if config.Command == "" {
			return errors.New("missing command")
		}
		if config.Name == nil {
			return errors.New("missing name")
		}

		return nil
	}, configs...)...)
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

		relativeCwd := filepath.Join(dir, fun.Deref(config.Cwd))
		cwd, err := filepath.Abs(relativeCwd)
		if err != nil {
			return fun.Zero[RunConfig](), errors.Wrapf(err, "get absolute cwd, relative is %q", relativeCwd)
		}

		args, err := fun.MapErr[string](func(arg any) (string, error) {
			switch a := arg.(type) {
			case fmt.Stringer:
				return a.String(), nil
			case int, int8, int16, int32, int64,
				uint, uint8, uint16, uint32, uint64,
				float32, float64, bool, string:
				return fmt.Sprint(arg), nil
			default:
				return "", errors.Newf("unknown arg type %T", arg)
			}
		}, config.Args...)
		if err != nil {
			return fun.Zero[RunConfig](), errors.Wrap(err, "parse args")
		}

		return RunConfig{
			Name:        *config.Name,
			Command:     config.Command,
			Args:        args,
			Tags:        config.Tags,
			Cwd:         cwd,
			Env:         config.Env,
			Watch:       watch,
			StdoutFile:  fun.Zero[fun.Option[string]](),
			StderrFile:  fun.Zero[fun.Option[string]](),
			KillTimeout: 0,
			Autorestart: false,
			MaxRestarts: 0,
			Startup:     config.Startup,
			DependsOn:   config.DependsOn,
			Cron:        fun.FromPtr(config.Cron),
		}, nil
	}, configs...)
}

func getStringPtr(section *ini.Section, key string) *string {
	if value := section.Key(key).String(); value != "" {
		return &value
	}
	return nil
}

func getStringSlice(section *ini.Section, key string) []string {
	if value := section.Key(key).String(); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result
	}
	return nil
}

func getArgsSlice(section *ini.Section) []any {
	var args []any
	if value := section.Key("args").String(); value != "" {
		parts := strings.Split(value, ",")
		for _, part := range parts {
			args = append(args, strings.TrimSpace(part))
		}
	}
	return args
}

func getEnvMap(section *ini.Section) map[string]string {
	env := make(map[string]string)
	for _, key := range section.Keys() {
		if strings.HasPrefix(key.Name(), "env.") {
			envName := strings.TrimPrefix(key.Name(), "env.")
			env[envName] = key.String()
		}
	}
	return env
}
