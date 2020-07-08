package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/errors"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/go-playground/validator.v9"
)

const defaultConfigFile = "athens.toml"

// Config provides configuration values for all components
type Config struct {
	TimeoutConf
	GoEnv            string    `validate:"required" envconfig:"GO_ENV"`
	GoBinary         string    `validate:"required" envconfig:"GO_BINARY_PATH"`
	GoProxy          string    `envconfig:"GOPROXY"`
	GoBinaryEnvVars  EnvList   `envconfig:"ATHENS_GO_BINARY_ENV_VARS"`
	GoGetWorkers     int       `validate:"required" envconfig:"ATHENS_GOGET_WORKERS"`
	GoGetDir         string    `envconfig:"ATHENS_GOGOET_DIR"`
	ProtocolWorkers  int       `validate:"required" envconfig:"ATHENS_PROTOCOL_WORKERS"`
	LogLevel         string    `validate:"required" envconfig:"ATHENS_LOG_LEVEL"`
	CloudRuntime     string    `validate:"required" envconfig:"ATHENS_CLOUD_RUNTIME"`
	EnablePprof      bool      `envconfig:"ATHENS_ENABLE_PPROF"`
	PprofPort        string    `envconfig:"ATHENS_PPROF_PORT"`
	FilterFile       string    `envconfig:"ATHENS_FILTER_FILE"`
	TraceExporterURL string    `envconfig:"ATHENS_TRACE_EXPORTER_URL"`
	TraceExporter    string    `envconfig:"ATHENS_TRACE_EXPORTER"`
	StatsExporter    string    `envconfig:"ATHENS_STATS_EXPORTER"`
	StorageType      string    `validate:"required" envconfig:"ATHENS_STORAGE_TYPE"`
	GlobalEndpoint   string    `envconfig:"ATHENS_GLOBAL_ENDPOINT"` // This feature is not yet implemented
	Port             string    `envconfig:"ATHENS_PORT"`
	BasicAuthUser    string    `envconfig:"BASIC_AUTH_USER"`
	BasicAuthPass    string    `envconfig:"BASIC_AUTH_PASS"`
	ForceSSL         bool      `envconfig:"PROXY_FORCE_SSL"`
	ValidatorHook    string    `envconfig:"ATHENS_PROXY_VALIDATOR"`
	PathPrefix       string    `envconfig:"ATHENS_PATH_PREFIX"`
	NETRCPath        string    `envconfig:"ATHENS_NETRC_PATH"`
	GithubToken      string    `envconfig:"ATHENS_GITHUB_TOKEN"`
	HGRCPath         string    `envconfig:"ATHENS_HGRC_PATH"`
	TLSCertFile      string    `envconfig:"ATHENS_TLSCERT_FILE"`
	TLSKeyFile       string    `envconfig:"ATHENS_TLSKEY_FILE"`
	SumDBs           []string  `envconfig:"ATHENS_SUM_DBS"`
	NoSumPatterns    []string  `envconfig:"ATHENS_GONOSUM_PATTERNS"`
	DownloadMode     mode.Mode `envconfig:"ATHENS_DOWNLOAD_MODE"`
	DownloadURL      string    `envconfig:"ATHENS_DOWNLOAD_URL"`
	SingleFlightType string    `envconfig:"ATHENS_SINGLE_FLIGHT_TYPE"`
	RobotsFile       string    `envconfig:"ATHENS_ROBOTS_FILE"`
	IndexType        string    `envconfig:"ATHENS_INDEX_TYPE"`
	SingleFlight     *SingleFlight
	Storage          *Storage
	Index            *Index
}

// EnvList is a list of key-value environment
// variables that are passed to the Go command
type EnvList []string

// HasKey returns whether a key-value entry
// is present by only checking the left of
// key=value
func (el EnvList) HasKey(key string) bool {
	for _, env := range el {
		if strings.HasPrefix(env, key+"=") {
			return true
		}
	}
	return false
}

// Add adds a key=value entry to the environment
// list
func (el *EnvList) Add(key, value string) {
	*el = append(*el, key+"="+value)
}

// Decode implements envconfig.Decoder. Please see the below link for more information on
// that interface:
//
// https://github.com/kelseyhightower/envconfig#custom-decoders
//
// We are doing this to allow for very long lists of assignments to be set inside of
// a single environment variable. For example:
//
//	ATHENS_GO_BINARY_ENV_VARS="GOPRIVATE=*.corp.example.com,rsc.io/private; GOPROXY=direct"
//
// See the below link for more information:
// https://github.com/gomods/athens/issues/1404
func (el *EnvList) Decode(value string) error {
	const op errors.Op = "envList.Decode"
	if value == "" {
		return nil
	}
	*el = EnvList{} // env vars must override config file
	assignments := strings.Split(value, ";")
	for _, assignment := range assignments {
		*el = append(*el, strings.TrimSpace(assignment))
	}
	return el.Validate()
}

// Validate validates that all strings inside the
// list are of the key=value format
func (el EnvList) Validate() error {
	const op errors.Op = "EnvList.Validate"
	for _, env := range el {
		// some strings can have multiple "=", such as GODEBUG=netdns=cgo
		if strings.Count(env, "=") < 1 {
			return errors.E(op, fmt.Errorf("incorrect env format: %v", env))
		}
	}
	return nil
}

// Load loads the config from a file.
// If file is not present returns default config
func Load(configFile string) (*Config, error) {
	// User explicitly specified a config file
	if configFile != "" {
		return ParseConfigFile(configFile)
	}

	// There is a config in the current directory
	if fi, err := os.Stat(defaultConfigFile); err == nil {
		return ParseConfigFile(fi.Name())
	}

	// Use default values
	log.Println("Running dev mode with default settings, consult config when you're ready to run in production")
	cfg := DefaultConfig()
	return cfg, envOverride(cfg)
}

// DefaultConfig returns default config
func DefaultConfig() *Config {
	return &Config{
		GoBinary:         "go",
		GoBinaryEnvVars:  EnvList{"GOPROXY=direct"},
		GoEnv:            "development",
		GoProxy:          "direct",
		GoGetWorkers:     10,
		ProtocolWorkers:  30,
		LogLevel:         "debug",
		CloudRuntime:     "none",
		EnablePprof:      false,
		PprofPort:        ":3001",
		StatsExporter:    "prometheus",
		TimeoutConf:      TimeoutConf{Timeout: 300},
		StorageType:      "memory",
		Port:             ":3000",
		SingleFlightType: "memory",
		GlobalEndpoint:   "http://localhost:3001",
		TraceExporterURL: "http://localhost:14268",
		SumDBs:           []string{"https://sum.golang.org"},
		NoSumPatterns:    []string{},
		DownloadMode:     "sync",
		DownloadURL:      "",
		RobotsFile:       "robots.txt",
		IndexType:        "none",
		SingleFlight: &SingleFlight{
			Etcd:  &Etcd{"localhost:2379,localhost:22379,localhost:32379"},
			Redis: &Redis{"127.0.0.1:6379", ""},
			RedisSentinel: &RedisSentinel{
				Endpoints:        []string{"127.0.0.1:26379"},
				MasterName:       "redis-1",
				SentinelPassword: "sekret",
			},
		},
		Index: &Index{
			MySQL: &MySQL{
				Protocol: "tcp",
				Host:     "localhost",
				Port:     3306,
				User:     "root",
				Password: "",
				Database: "athens",
				Params: map[string]string{
					"parseTime": "true",
					"timeout":   "30s",
				},
			},
			Postgres: &Postgres{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "",
				Database: "athens",
				Params: map[string]string{
					"connect_timeout": "30",
					"sslmode":         "disable",
				},
			},
		},
	}
}

// BasicAuth returns BasicAuthUser and BasicAuthPassword
// and ok if neither of them are empty
func (c *Config) BasicAuth() (user, pass string, ok bool) {
	user = c.BasicAuthUser
	pass = c.BasicAuthPass
	ok = user != "" && pass != ""
	return user, pass, ok
}

// TLSCertFiles returns certificate and key files and an error if
// both files doesn't exist and have approperiate file permissions
func (c *Config) TLSCertFiles() (cert, key string, err error) {
	if c.TLSCertFile == "" && c.TLSKeyFile == "" {
		return "", "", nil
	}

	certFile, err := os.Stat(c.TLSCertFile)
	if err != nil {
		return "", "", fmt.Errorf("Could not access TLSCertFile: %v", err)
	}

	keyFile, err := os.Stat(c.TLSKeyFile)
	if err != nil {
		return "", "", fmt.Errorf("Could not access TLSKeyFile: %v", err)
	}

	if keyFile.Mode()&077 != 0 && runtime.GOOS != "windows" {
		return "", "", fmt.Errorf("TLSKeyFile should not be accessible by others")
	}

	return certFile.Name(), keyFile.Name(), nil
}

// FilterOff returns true if the FilterFile is empty
func (c *Config) FilterOff() bool {
	return c.FilterFile == ""
}

// ParseConfigFile parses the given file into an athens config struct
func ParseConfigFile(configFile string) (*Config, error) {

	var config Config
	// attempt to read the given config file
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, err
	}

	// override values with environment variables if specified
	if err := envOverride(&config); err != nil {
		return nil, err
	}

	// Check file perms from config
	if config.GoEnv == "production" {
		if err := checkFilePerms(configFile, config.FilterFile); err != nil {
			return nil, err
		}
	}

	// validate all required fields have been populated
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return &config, nil
}

// envOverride uses Environment variables to override unspecified properties
func envOverride(config *Config) error {
	const defaultPort = ":3000"
	err := envconfig.Process("athens", config)
	if err != nil {
		return err
	}
	portEnv := os.Getenv("PORT")
	// ATHENS_PORT takes precedence over PORT
	if portEnv != "" && os.Getenv("ATHENS_PORT") == "" {
		config.Port = portEnv
	}
	if config.Port == "" {
		config.Port = defaultPort
	}
	config.Port = ensurePortFormat(config.Port)
	return nil
}

func ensurePortFormat(s string) string {
	if _, err := strconv.Atoi(s); err == nil {
		return ":" + s
	}
	return s
}

func validateConfig(config Config) error {
	validate := validator.New()
	err := validate.StructExcept(config, "Storage", "Index")
	if err != nil {
		return err
	}
	err = validateStorage(validate, config.StorageType, config.Storage)
	if err != nil {
		return err
	}
	err = validateIndex(validate, config.IndexType, config.Index)
	if err != nil {
		return err
	}
	return nil
}

func validateStorage(validate *validator.Validate, storageType string, config *Storage) error {
	switch storageType {
	case "memory":
		return nil
	case "mongo":
		return validate.Struct(config.Mongo)
	case "disk":
		return validate.Struct(config.Disk)
	case "minio":
		return validate.Struct(config.Minio)
	case "gcp":
		return validate.Struct(config.GCP)
	case "s3":
		return validate.Struct(config.S3)
	case "azureblob":
		return validate.Struct(config.AzureBlob)
	case "external":
		return validate.Struct(config.External)
	default:
		return fmt.Errorf("storage type %q is unknown", storageType)
	}
}

func validateIndex(validate *validator.Validate, indexType string, config *Index) error {
	switch indexType {
	case "", "none", "memory":
		return nil
	case "mysql":
		return validate.Struct(config.MySQL)
	case "postgres":
		return validate.Struct(config.Postgres)
	default:
		return fmt.Errorf("index type %q is unknown", indexType)
	}
}

// GetConf accepts the path to a file, constructs an absolute path to the file,
// and attempts to parse it into a Config struct.
func GetConf(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("Unable to construct absolute path to test config file")
	}
	conf, err := ParseConfigFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse test config file: %s", err.Error())
	}
	return conf, nil
}

// checkFilePerms given a list of files
func checkFilePerms(files ...string) error {
	const op = "config.checkFilePerms"

	for _, f := range files {
		if f == "" {
			continue
		}

		// TODO: Do not ignore errors when a file is not found
		// There is a subtle bug in the filter module which ignores the filter file if it does not find it.
		// This check can be removed once that has been fixed
		fInfo, err := os.Stat(f)
		if err != nil {
			continue
		}

		// Assume unix based system (MacOS and Linux)
		// the bit mask is calculated using the umask command which tells which permissions
		// should not be allowed for a particular user, group or world
		if fInfo.Mode()&0077 != 0 && runtime.GOOS != "windows" {
			return errors.E(op, f+" should have at most rwx,-, - (bit mask 077) as permission")
		}

	}

	return nil
}
