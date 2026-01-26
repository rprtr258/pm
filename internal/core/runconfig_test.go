package core

import (
	"cmp"
	"embed"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/rprtr258/fun"
	"github.com/shoenig/test/must"
)

//go:embed testdata
var testdataFS embed.FS

func useTestDataFile(t *testing.T, filename string) string {
	t.Helper()
	data, err := testdataFS.ReadFile(filepath.Join("testdata", filename))
	must.NoError(t, err)
	return string(data)
}

func useTmpConfigFile(t *testing.T, filename, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, filename)
	must.NoError(t, os.WriteFile(configFile, []byte(content), 0o644))
	return configFile
}

func useLoadConfigs(t *testing.T, configFile string) []RunConfig {
	t.Helper()
	configs, err := LoadConfigs(configFile)
	must.NoError(t, err)
	return configs
}

func assertConfig(t *testing.T, expected, actual []RunConfig) {
	t.Helper()
	must.Eq(t, len(expected), len(actual))
	slices.SortFunc(expected, func(a, b RunConfig) int { return cmp.Compare(a.Name, b.Name) })
	slices.SortFunc(actual, func(a, b RunConfig) int { return cmp.Compare(a.Name, b.Name) })
	for i, config := range expected {
		must.Eq(t, config.Watch.Valid, actual[i].Watch.Valid)
		if config.Watch.Valid {
			must.Eq(t, config.Watch.Value.String(), actual[i].Watch.Value.String())
			// NOTE: regexp removal
			config.Watch.Value = nil
			actual[i].Watch.Value = nil
		}
		actual[i].Cwd = "" // NOTE: temp dir removal
		must.Eq(t, config, actual[i])
	}
}

func TestDocsExamples(t *testing.T) {
	t.Parallel()
	exampleConfig := RunConfig{
		Name:    "web-server",
		Command: "node",
		Args:    []string{"server.js"},
		Env:     map[string]string{"PORT": "3000", "NODE_ENV": "production"},
		Tags:    []string{"web"},
		Startup: true,
	}
	for name, filename := range map[string]string{
		"JSONNet": "config.jsonnet",
		"YAML":    "config.yaml",
		"TOML":    "config.toml",
		"INI":     "config.ini",
		"HCL":     "config.hcl",
		"JSON":    "config.json",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", filename))
			must.NoError(t, err)
			setup := useTmpConfigFile(t, filename, string(data))
			configs := useLoadConfigs(t, setup)
			assertConfig(t, []RunConfig{exampleConfig}, configs)
		})
	}
}

func TestLoadConfigs_Formats(t *testing.T) {
	t.Parallel()
	tests := map[string][]RunConfig{
		"yaml_relative_cwd.yaml": {
			{
				Name:    "relative-cwd-test",
				Command: "echo",
				Args:    []string{},
			},
		},
		"yaml_multi_process.yaml": {
			{
				Name:      "test-process",
				Command:   "echo",
				Args:      []string{"hello", "world"},
				Env:       map[string]string{"TEST_VAR": "test_value", "ANOTHER_VAR": "another_value"},
				Tags:      []string{"test", "demo"},
				Startup:   true,
				DependsOn: []string{"db", "cache"},
				Watch:     fun.Valid(regexp.MustCompile(`.*\.go`)),
				Cron:      fun.Valid("0 */6 * * *"),
			},
			{
				Name:    "sleep-process",
				Command: "sleep",
				Args:    []string{"10"},
			},
		},
		"ini_multi_process.ini": {
			{
				Name:      "database",
				Command:   "postgres",
				Args:      []string{"-D", "/var/lib/postgresql/data"},
				Env:       map[string]string{"DB_HOST": "localhost", "DB_PORT": "5432"},
				Tags:      []string{"database", "storage"},
				Startup:   true,
				DependsOn: []string{"network"},
			},
			{
				Name:    "web-server",
				Command: "nginx",
				Args:    []string{"-c", "/etc/nginx/nginx.conf"},
				Env:     map[string]string{"PORT": "80"},
				Tags:    []string{"web"},
			},
		},
		"hcl_multi_process.hcl": {
			{
				Name:      "database",
				Command:   "postgres",
				Args:      []string{"-D", "/var/lib/postgresql/data"},
				Env:       map[string]string{"DB_HOST": "localhost", "DB_PORT": "5432"},
				Tags:      []string{"database", "storage"},
				Startup:   true,
				DependsOn: []string{"network"},
			},
			{
				Name:    "web-server",
				Command: "nginx",
				Args:    []string{"-c", "/etc/nginx/nginx.conf"},
				Env:     map[string]string{"PORT": "80"},
				Tags:    []string{"web"},
			},
		},
		"json_simple.json": {
			{
				Name:    "simple-service",
				Command: "echo",
				Args:    []string{"Hello", "JSON!"},
				Env:     map[string]string{"SERVICE_NAME": "test", "VERSION": "1.0.0"},
				Tags:    []string{"simple", "test"},
			},
		},
		"toml_simple.toml": {
			{
				Name:    "simple-service",
				Command: "echo",
				Args:    []string{"Hello", "TOML!"},
				Env:     map[string]string{"SERVICE_NAME": "test", "VERSION": "1.0.0"},
				Tags:    []string{"simple", "test"},
			},
		},
		"yaml_simple.yaml": {
			{
				Name:    "echo-first",
				Command: "echo",
				Args:    []string{"first"},
			},
			{
				Name:    "echo-second",
				Command: "echo",
				Args:    []string{"second"},
			},
		},
		"yaml_with_watch.yaml": {
			{
				Name:    "watch-test",
				Command: "echo",
				Args:    []string{},
				Watch:   fun.Valid(regexp.MustCompile(`.*\.(go|js|ts)$`)),
			},
		},
	}
	for testdataFile, expectedConfig := range tests {
		t.Run(testdataFile, func(t *testing.T) {
			t.Parallel()
			filename := "config" + filepath.Ext(testdataFile)
			setup := useTmpConfigFile(t, filename, useTestDataFile(t, testdataFile))
			configs := useLoadConfigs(t, setup)
			assertConfig(t, expectedConfig, configs)
		})
	}
}

func TestLoadConfigs_ValidationErrors(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		content    string
		configFile string
		errMsg     string
	}{
		"yaml_missing_command": {
			content:    useTestDataFile(t, "yaml_missing_command.yaml"),
			configFile: "config.yaml",
			errMsg:     "missing command",
		},
		"yaml_invalid_format": {
			content: `- command: "echo"
  args: ["test"]`,
			configFile: "config.yaml",
			errMsg:     "parse yaml",
		},
		"unsupported_format": {
			content:    `test`,
			configFile: "config.txt",
			errMsg:     `unsupported config format ".txt"`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			configFile := useTmpConfigFile(t, tt.configFile, tt.content)
			_, err := LoadConfigs(configFile)
			must.Error(t, err)
			must.ErrorContains(t, err, tt.errMsg)
		})
	}
}
