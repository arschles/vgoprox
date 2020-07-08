package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// GetServicePort returns the host and port for a service from docker-compose
func GetServicePort(t *testing.T, service string, containerPort int) (host string, hostPort int) {
	t.Helper()
	cPort := strconv.Itoa(containerPort)
	project := os.Getenv("DOCKER_COMPOSE_PROJECT")
	if project == "" {
		project = "athenstest"
	}
	out, err := exec.Command("docker-compose", "-p", project, "port", service, cPort).Output()
	require.NoError(t, err)
	addr := strings.TrimSpace(string(out))
	parts := strings.Split(addr, ":")
	require.Lenf(t, parts, 2, "invalid address %q", addr)
	hPort, err := strconv.Atoi(parts[1])
	require.NoError(t, err)
	return parts[0], hPort
}

// AthensRoot returns the filepath to the root of this repository
func AthensRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	file = filepath.Dir(file)
	file = filepath.Dir(file)
	file = filepath.Dir(file)
	return file
}

type TestDependency int

const (
	TestDependencyMySQL TestDependency = iota
	TestDependencyPostgres
	TestDependencyRedis
	TestDependencyProtectedRedis
	TestDependencyEtcd
	TestDependencyMinio
	TestDependencyMongo
	TestDependencyAzurite
	invalidDependency // keep this at the end so we can iterate through dependencies
)

var dependencyNames = map[TestDependency]string{
	TestDependencyMySQL:          "mysql",
	TestDependencyPostgres:       "postgres",
	TestDependencyRedis:          "redis",
	TestDependencyProtectedRedis: "protectedredis",
	TestDependencyEtcd:           "etcd",
	TestDependencyMinio:          "minio",
	TestDependencyMongo:          "mongo",
	TestDependencyAzurite:        "azurite",
}

var dependencySkipVars = map[TestDependency]string{
	TestDependencyMySQL:          "SKIP_MYSQL",
	TestDependencyPostgres:       "SKIP_POSTGRES",
	TestDependencyRedis:          "SKIP_REDIS",
	TestDependencyProtectedRedis: "SKIP_PROTECTEDREDIS",
	TestDependencyEtcd:           "SKIP_ETCD",
	TestDependencyMinio:          "SKIP_MINIO",
	TestDependencyMongo:          "SKIP_MONGO",
	TestDependencyAzurite:        "SKIP_AZURITE",
}

func (d TestDependency) String() string {
	return dependencyNames[d]
}

func (d TestDependency) Check(t testing.TB) {
	if os.Getenv(dependencySkipVars[d]) != "" {
		t.Skipf("skipping test because %s dependency is not met", d)
	}
}

func CheckTestDependencies(t testing.TB, dependencies ...TestDependency) {
	if os.Getenv("SKIP_ALL_DEPENDENCIES") != "" {
		t.SkipNow()
	}
	for _, dependency := range dependencies {
		dependency.Check(t)
	}
}
