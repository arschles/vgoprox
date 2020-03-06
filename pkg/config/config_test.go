package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func testConfigFile(t *testing.T) (testConfigFile string) {
	testConfigFile = filepath.Join("..", "..", "config.dev.toml")
	if err := os.Chmod(testConfigFile, 0700); err != nil {
		t.Fatalf("%s\n", err)
	}
	return testConfigFile
}

func compareConfigs(parsedConf *Config, expConf *Config, t *testing.T) {
	opts := cmpopts.IgnoreTypes(StorageConfig{}, SingleFlight{})
	eq := cmp.Equal(parsedConf, expConf, opts)
	if !eq {
		t.Errorf("Parsed Example configuration did not match expected values. Expected: %+v. Actual: %+v", expConf, parsedConf)
	}
}

func compareStorageConfigs(parsedStorage *StorageConfig, expStorage *StorageConfig, t *testing.T) {
	eq := cmp.Equal(parsedStorage.Mongo, expStorage.Mongo)
	if !eq {
		t.Errorf("Parsed Example Storage configuration did not match expected values. Expected: %+v. Actual: %+v", expStorage.Mongo, parsedStorage.Mongo)
	}
	eq = cmp.Equal(parsedStorage.Minio, expStorage.Minio)
	if !eq {
		t.Errorf("Parsed Example Storage configuration did not match expected values. Expected: %+v. Actual: %+v", expStorage.Minio, parsedStorage.Minio)
	}
	eq = cmp.Equal(parsedStorage.Disk, expStorage.Disk)
	if !eq {
		t.Errorf("Parsed Example Storage configuration did not match expected values. Expected: %+v. Actual: %+v", expStorage.Disk, parsedStorage.Disk)
	}
	eq = cmp.Equal(parsedStorage.GCP, expStorage.GCP)
	if !eq {
		t.Errorf("Parsed Example Storage configuration did not match expected values. Expected: %+v. Actual: %+v", expStorage.GCP, parsedStorage.GCP)
	}
	eq = cmp.Equal(parsedStorage.S3, expStorage.S3)
	if !eq {
		t.Errorf("Parsed Example Storage configuration did not match expected values. Expected: %+v. Actual: %+v", expStorage.S3, parsedStorage.S3)
	}
}

func TestPortDefaultsCorrectly(t *testing.T) {
	conf := &Config{}
	err := envOverride(conf)
	if err != nil {
		t.Fatalf("Env override failed: %v", err)
	}
	expPort := ":3000"
	if conf.Port != expPort {
		t.Errorf("Port was incorrect. Got: %s, want: %s", conf.Port, expPort)
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Clearenv()
	expConf := &Config{
		GoEnv:           "production",
		GoGetWorkers:    10,
		ProtocolWorkers: 10,
		LogLevel:        "info",
		GoBinary:        "go11",
		CloudRuntime:    "gcp",
		TimeoutConf: TimeoutConf{
			Timeout: 30,
		},
		StorageType:     "minio",
		GlobalEndpoint:  "mytikas.gomods.io",
		Port:            ":7000",
		EnablePprof:     false,
		PprofPort:       ":3001",
		BasicAuthUser:   "testuser",
		BasicAuthPass:   "testpass",
		ForceSSL:        true,
		ValidatorHook:   "testhook.io",
		PathPrefix:      "prefix",
		NETRCPath:       "/test/path/.netrc",
		HGRCPath:        "/test/path/.hgrc",
		Storage:         &StorageConfig{},
		GoBinaryEnvVars: []string{"GOPROXY=direct"},
		SingleFlight:    &SingleFlight{},
		RobotsFile:      "robots.txt",
	}

	envVars := getEnvMap(expConf)
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	conf := &Config{}
	err := envOverride(conf)
	if err != nil {
		t.Fatalf("Env override failed: %v", err)
	}
	compareConfigs(conf, expConf, t)
}

func TestEnvOverridesPreservingPort(t *testing.T) {
	os.Clearenv()
	const expPort = ":5000"
	conf := &Config{Port: expPort}
	err := envOverride(conf)
	if err != nil {
		t.Fatalf("Env override failed: %v", err)
	}
	if conf.Port != expPort {
		t.Errorf("Port was incorrect. Got: %s, want: %s", conf.Port, expPort)
	}
}

func TestEnvOverridesPORT(t *testing.T) {
	conf := &Config{Port: ""}
	oldVal := os.Getenv("PORT")
	defer os.Setenv("PORT", oldVal)
	os.Setenv("PORT", "5000")
	err := envOverride(conf)
	if err != nil {
		t.Fatalf("Env override failed: %v", err)
	}
	if conf.Port != ":5000" {
		t.Fatalf("expected PORT env to be :5000 but got %v", conf.Port)
	}
}

func TestEnsurePortFormat(t *testing.T) {
	port := "3000"
	expected := ":3000"
	given := ensurePortFormat(port)
	if given != expected {
		t.Fatalf("expected ensurePortFormat to add a colon to %v but got %v", port, given)
	}
	port = ":3000"
	given = ensurePortFormat(port)
	if given != expected {
		t.Fatalf("expected ensurePortFormat to not add a colon when it's present but got %v", given)
	}
	port = "127.0.0.1:3000"
	expected = "127.0.0.1:3000"
	given = ensurePortFormat(port)
	if given != expected {
		t.Fatalf("expected ensurePortFormat to not add a colon when it's present but got %v", given)
	}
}

func TestStorageEnvOverrides(t *testing.T) {
	expStorage := &StorageConfig{
		Disk: &DiskConfig{
			RootPath: "/my/root/path",
		},
		GCP: &GCPConfig{
			ProjectID: "gcpproject",
			Bucket:    "gcpbucket",
		},
		Minio: &MinioConfig{
			Endpoint:  "minioEndpoint",
			Key:       "minioKey",
			Secret:    "minioSecret",
			EnableSSL: false,
			Bucket:    "minioBucket",
			Region:    "us-west-1",
		},
		Mongo: &MongoConfig{
			URL:                   "mongoURL",
			CertPath:              "/test/path",
			DefaultDBName:         "test",
			DefaultCollectionName: "testModules",
		},
		S3: &S3Config{
			Region: "s3Region",
			Key:    "s3Key",
			Secret: "s3Secret",
			Token:  "s3Token",
			Bucket: "s3Bucket",
		},
	}
	envVars := getEnvMap(&Config{Storage: expStorage})
	envVarBackup := map[string]string{}
	for k, v := range envVars {
		oldVal := os.Getenv(k)
		envVarBackup[k] = oldVal
		os.Setenv(k, v)
	}
	conf := &Config{}
	err := envOverride(conf)
	if err != nil {
		t.Fatalf("Env override failed: %v", err)
	}
	compareStorageConfigs(conf.Storage, expStorage, t)
	restoreEnv(envVarBackup)
}

// TestParseExampleConfig validates that all the properties in the example configuration file
// can be parsed and validated without any environment variables
func TestParseExampleConfig(t *testing.T) {

	// initialize all struct pointers so we get all applicable env variables
	emptyConf := &Config{
		Storage: &StorageConfig{
			Disk: &DiskConfig{},
			GCP:  &GCPConfig{},
			Minio: &MinioConfig{
				EnableSSL: false,
			},
			Mongo: &MongoConfig{},
			S3:    &S3Config{},
		},
		SingleFlight: &SingleFlight{},
	}
	// unset all environment variables
	envVars := getEnvMap(emptyConf)
	envVarBackup := map[string]string{}
	for k := range envVars {
		oldVal := os.Getenv(k)
		envVarBackup[k] = oldVal
		os.Unsetenv(k)
	}

	expStorage := &StorageConfig{
		Disk: &DiskConfig{
			RootPath: "/path/on/disk",
		},
		GCP: &GCPConfig{
			ProjectID: "MY_GCP_PROJECT_ID",
			Bucket:    "MY_GCP_BUCKET",
		},
		Minio: &MinioConfig{
			Endpoint:  "127.0.0.1:9001",
			Key:       "minio",
			Secret:    "minio123",
			EnableSSL: false,
			Bucket:    "gomods",
		},
		Mongo: &MongoConfig{
			URL:                   "mongodb://127.0.0.1:27017",
			CertPath:              "",
			InsecureConn:          false,
			DefaultDBName:         "athens",
			DefaultCollectionName: "modules",
		},
		S3: &S3Config{
			Region: "MY_AWS_REGION",
			Key:    "MY_AWS_ACCESS_KEY_ID",
			Secret: "MY_AWS_SECRET_ACCESS_KEY",
			Token:  "",
			Bucket: "MY_S3_BUCKET_NAME",
		},
	}

	expConf := &Config{
		GoEnv:           "development",
		LogLevel:        "debug",
		GoBinary:        "go",
		GoGetWorkers:    10,
		ProtocolWorkers: 30,
		CloudRuntime:    "none",
		TimeoutConf: TimeoutConf{
			Timeout: 300,
		},
		StorageType:      "memory",
		GlobalEndpoint:   "http://localhost:3001",
		Port:             ":3000",
		EnablePprof:      false,
		PprofPort:        ":3001",
		BasicAuthUser:    "",
		BasicAuthPass:    "",
		Storage:          expStorage,
		TraceExporterURL: "http://localhost:14268",
		TraceExporter:    "",
		StatsExporter:    "prometheus",
		SingleFlightType: "memory",
		GoBinaryEnvVars:  []string{"GOPROXY=direct"},
		SingleFlight:     &SingleFlight{},
		SumDBs:           []string{"https://sum.golang.org"},
		NoSumPatterns:    []string{},
		DownloadMode:     "sync",
		RobotsFile:       "robots.txt",
	}

	absPath, err := filepath.Abs(testConfigFile(t))
	if err != nil {
		t.Errorf("Unable to construct absolute path to example config file")
	}
	parsedConf, err := ParseConfigFile(absPath)
	if err != nil {
		t.Errorf("Unable to parse example config file: %+v", err)
	}

	compareConfigs(parsedConf, expConf, t)
	restoreEnv(envVarBackup)
}

func getEnvMap(config *Config) map[string]string {

	envVars := map[string]string{
		"GO_ENV":                  config.GoEnv,
		"GO_BINARY_PATH":          config.GoBinary,
		"ATHENS_GOGET_WORKERS":    strconv.Itoa(config.GoGetWorkers),
		"ATHENS_PROTOCOL_WORKERS": strconv.Itoa(config.ProtocolWorkers),
		"ATHENS_LOG_LEVEL":        config.LogLevel,
		"ATHENS_CLOUD_RUNTIME":    config.CloudRuntime,
		"ATHENS_TIMEOUT":          strconv.Itoa(config.Timeout),
	}

	envVars["ATHENS_STORAGE_TYPE"] = config.StorageType
	envVars["ATHENS_GLOBAL_ENDPOINT"] = config.GlobalEndpoint
	envVars["ATHENS_PORT"] = config.Port
	envVars["ATHENS_ENABLE_PPROF"] = strconv.FormatBool(config.EnablePprof)
	envVars["ATHENS_PPROF_PORT"] = config.PprofPort
	envVars["BASIC_AUTH_USER"] = config.BasicAuthUser
	envVars["BASIC_AUTH_PASS"] = config.BasicAuthPass
	envVars["PROXY_FORCE_SSL"] = strconv.FormatBool(config.ForceSSL)
	envVars["ATHENS_PROXY_VALIDATOR"] = config.ValidatorHook
	envVars["ATHENS_PATH_PREFIX"] = config.PathPrefix
	envVars["ATHENS_NETRC_PATH"] = config.NETRCPath
	envVars["ATHENS_HGRC_PATH"] = config.HGRCPath
	envVars["ATHENS_ROBOTS_FILE"] = config.RobotsFile
	envVars["ATHENS_GO_BINARY_ENV_VARS"] = strings.Join(config.GoBinaryEnvVars, ",")

	storage := config.Storage
	if storage != nil {
		if storage.Disk != nil {
			envVars["ATHENS_DISK_STORAGE_ROOT"] = storage.Disk.RootPath
		}
		if storage.GCP != nil {
			envVars["GOOGLE_CLOUD_PROJECT"] = storage.GCP.ProjectID
			envVars["ATHENS_STORAGE_GCP_BUCKET"] = storage.GCP.Bucket
		}
		if storage.Minio != nil {
			envVars["ATHENS_MINIO_ENDPOINT"] = storage.Minio.Endpoint
			envVars["ATHENS_MINIO_ACCESS_KEY_ID"] = storage.Minio.Key
			envVars["ATHENS_MINIO_SECRET_ACCESS_KEY"] = storage.Minio.Secret
			envVars["ATHENS_MINIO_USE_SSL"] = strconv.FormatBool(storage.Minio.EnableSSL)
			envVars["ATHENS_MINIO_REGION"] = storage.Minio.Region
			envVars["ATHENS_MINIO_BUCKET_NAME"] = storage.Minio.Bucket
		}
		if storage.Mongo != nil {
			envVars["ATHENS_MONGO_STORAGE_URL"] = storage.Mongo.URL
			envVars["ATHENS_MONGO_CERT_PATH"] = storage.Mongo.CertPath
			envVars["ATHENS_MONGO_INSECURE"] = strconv.FormatBool(storage.Mongo.InsecureConn)
			envVars["ATHENS_MONGO_DEFAULT_DATABASE"] = storage.Mongo.DefaultDBName
			envVars["ATHENS_MONGO_DEFAULT_COLLECTION"] = storage.Mongo.DefaultCollectionName

		}
		if storage.S3 != nil {
			envVars["AWS_REGION"] = storage.S3.Region
			envVars["AWS_ACCESS_KEY_ID"] = storage.S3.Key
			envVars["AWS_SECRET_ACCESS_KEY"] = storage.S3.Secret
			envVars["AWS_SESSION_TOKEN"] = storage.S3.Token
			envVars["AWS_FORCE_PATH_STYLE"] = strconv.FormatBool(storage.S3.ForcePathStyle)
			envVars["ATHENS_S3_BUCKET_NAME"] = storage.S3.Bucket
		}
	}
	return envVars
}

func restoreEnv(envVars map[string]string) {
	for k, v := range envVars {
		if v != "" {
			os.Setenv(k, v)
		} else {
			os.Unsetenv(k)
		}
	}
}

func tempFile(perm os.FileMode) (name string, err error) {
	f, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		return "", err
	}
	if err = os.Chmod(f.Name(), perm); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func Test_checkFilePerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skipf("Chmod is not supported in windows, so not possible to test. Ref: https://github.com/golang/go/blob/master/src/os/os_test.go#L1031\n")
	}

	incorrectPerms := []os.FileMode{0777, 0610, 0660}
	var incorrectFiles = make([]string, len(incorrectPerms))

	for i := range incorrectPerms {
		f, err := tempFile(incorrectPerms[i])
		if err != nil {
			t.Fatalf("tempFile creation error %s", err)
		}
		incorrectFiles[i] = f
		defer os.Remove(f)
	}

	correctPerms := []os.FileMode{0600, 0400}
	var correctFiles = make([]string, len(correctPerms))

	for i := range correctPerms {
		f, err := tempFile(correctPerms[i])
		if err != nil {
			t.Fatalf("tempFile creation error %s", err)
		}
		correctFiles[i] = f
		defer os.Remove(f)
	}

	type test struct {
		name    string
		files   []string
		wantErr bool
	}

	tests := []test{
		{
			"should not have an error on 0600, 0400, 0640",
			[]string{correctFiles[0], correctFiles[1]},
			false,
		},
		{
			"should not have an error on empty file name",
			[]string{"", correctFiles[1]},
			false,
		},
		{
			"should have an error if all the files have incorrect permissions",
			[]string{incorrectFiles[0], incorrectFiles[1], incorrectFiles[1]},
			true,
		},
		{
			"should have an error when at least 1 file has wrong permissions",
			[]string{correctFiles[0], correctFiles[1], incorrectFiles[1]},
			true,
		},
	}

	for _, f := range incorrectFiles {
		tests = append(tests, test{
			"incorrect file permission passed",
			[]string{f},
			true,
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkFilePerms(tt.files...); (err != nil) != tt.wantErr {
				t.Errorf("checkFilePerms() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfigMatchesConfigFile(t *testing.T) {
	absPath, err := filepath.Abs(testConfigFile(t))
	if err != nil {
		t.Errorf("Unable to construct absolute path to example config file")
	}
	parsedConf, err := ParseConfigFile(absPath)
	if err != nil {
		t.Errorf("Unable to parse example config file: %+v", err)
	}

	defConf := defaultConfig()

	ignoreStorageOpts := cmpopts.IgnoreTypes(&StorageConfig{})
	ignoreGoEnvOpts := cmpopts.IgnoreFields(Config{}, "GoEnv")
	eq := cmp.Equal(defConf, parsedConf, ignoreStorageOpts, ignoreGoEnvOpts)
	if !eq {
		t.Errorf("Default values from the config file: %v should equal to the default values returned in case the config file isn't provided %v", parsedConf, defConf)
	}
}

func TestEnvList(t *testing.T) {
	el := EnvList{"KEY=VALUE"}
	if !el.HasKey("KEY") {
		t.Fatal("expected KEY to be present")
	}
	if el.HasKey("KEY=") {
		t.Fatal("expected KEY= to not be found")
	}
	el.Add("HELLO", "WORLD")
	if !el.HasKey("HELLO") {
		t.Fatal("expected HELLO key to be found")
	}
	if err := el.Validate(); err != nil {
		t.Fatalf("expected err to be nil but got %v", err)
	}
	el = EnvList{"HELLO"}
	if err := el.Validate(); err == nil {
		t.Fatal("expected a validation error for incorrect formatting but got nil")
	}
	el = EnvList{"GODEBUG=netdns=cgo"}
	if !el.HasKey("GODEBUG") {
		t.Fatal("expected GODEBUG key to be present")
	}
	if el.HasKey("GODEBUG=") {
		t.Fatal("expected GODEBUG= key not to be found")
	}
	if err := el.Validate(); err != nil {
		t.Fatalf("expected err to be nil but got %v", err)
	}
}

type decodeTestCase struct {
	name     string
	pre      EnvList
	given    string
	valid    bool
	expected EnvList
}

var envListDecodeTests = []decodeTestCase{
	{
		name:     "empty",
		pre:      EnvList{},
		given:    "",
		valid:    true,
		expected: EnvList{},
	},
	{
		name:     "unchanged",
		pre:      EnvList{"GOPROXY=direct"},
		given:    "",
		valid:    true,
		expected: EnvList{"GOPROXY=direct"},
	},
	{
		name:     "must not merge",
		pre:      EnvList{"GOPROXY=direct"},
		given:    "GOPRIVATE=github.com/gomods/*",
		valid:    true,
		expected: EnvList{"GOPRIVATE=github.com/gomods/*"},
	},
	{
		name:     "must override",
		pre:      EnvList{"GOPROXY=direct"},
		given:    "GOPROXY=https://proxy.golang.org",
		valid:    true,
		expected: EnvList{"GOPROXY=https://proxy.golang.org"},
	},
	{
		name:  "semi colon separator",
		pre:   EnvList{"GOPROXY=direct", "GOPRIVATE="},
		given: "GOPROXY=off; GOPRIVATE=marwan.io/*;GONUTS=lol;GODEBUG=dns=true",
		valid: true,
		expected: EnvList{
			"GOPROXY=off",
			"GOPRIVATE=marwan.io/*",
			"GONUTS=lol",
			"GODEBUG=dns=true",
		},
	},
	{
		name:  "with commas",
		pre:   EnvList{"GOPROXY=direct", "GOPRIVATE="},
		given: "GOPROXY=proxy.golang.org,direct;GOPRIVATE=marwan.io/*;GONUTS=lol;GODEBUG=dns=true",
		valid: true,
		expected: EnvList{
			"GOPROXY=proxy.golang.org,direct",
			"GOPRIVATE=marwan.io/*",
			"GONUTS=lol",
			"GODEBUG=dns=true",
		},
	},
	{
		name:  "invalid",
		pre:   EnvList{},
		given: "GOPROXY=direct; INVALID",
		valid: false,
	},
	{
		name:     "accept empty value",
		pre:      EnvList{"GOPROXY=direct"},
		given:    "GOPROXY=; GOPRIVATE=github.com/*",
		valid:    true,
		expected: EnvList{"GOPROXY=", "GOPRIVATE=github.com/*"},
	},
}

func TestEnvListDecode(t *testing.T) {
	for _, tc := range envListDecodeTests {
		t.Run(tc.name, func(t *testing.T) {
			testDecode(t, tc)
		})
	}
	cfg := &Config{
		GoBinaryEnvVars: EnvList{"GOPROXY=direct"},
	}
	err := cfg.GoBinaryEnvVars.Decode("GOPROXY=https://proxy.golang.org; GOPRIVATE=github.com/gomods/*")
	if err != nil {
		t.Fatal(err)
	}
	cfg.GoBinaryEnvVars.Validate()
}

func testDecode(t *testing.T, tc decodeTestCase) {
	const envKey = "ATHENS_LIST_TEST"

	os.Setenv(envKey, tc.given)
	defer func() {
		require.NoError(t, os.Unsetenv(envKey))
	}()

	var config struct {
		GoBinaryEnvVars EnvList `envconfig:"ATHENS_LIST_TEST"`
	}
	config.GoBinaryEnvVars = tc.pre
	err := envconfig.Process("", &config)
	if tc.valid && err != nil {
		t.Fatal(err)
	}
	if !tc.valid {
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		return
	}
	require.Equal(t, tc.expected, config.GoBinaryEnvVars)
}
