package stash

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gomods/athens/internal/testutil"
	"github.com/gomods/athens/internal/testutil/testconfig"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/mem"
	"golang.org/x/sync/errgroup"
)

// WithRedisLock will ensure that 5 concurrent requests will all get the first request's
// response. We can ensure that because only the first response does not return an error
// and therefore all 5 responses should have no error.
func TestWithRedisLock(t *testing.T) {
	t.Parallel()
	testutil.CheckTestDependencies(t, testutil.TestDependencyRedis)
	cfg := testconfig.LoadTestConfig(t).SingleFlight.Redis
	strg, err := mem.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	ms := &mockRedisStasher{strg: strg}
	wrapper, err := WithRedisLock(cfg.Endpoint, cfg.Password, storage.WithChecker(strg))
	if err != nil {
		t.Fatal(err)
	}
	s := wrapper(ms)

	var eg errgroup.Group
	for i := 0; i < 5; i++ {
		eg.Go(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			_, err := s.Stash(ctx, "mod", "ver")
			return err
		})
	}

	err = eg.Wait()
	if err != nil {
		t.Fatal(err)
	}
}

// Verify with WithRedisLock working with password protected redis
// Same logic as the TestWithRedisLock test.
func TestWithRedisLockWithPassword(t *testing.T) {
	t.Parallel()
	testutil.CheckTestDependencies(t, testutil.TestDependencyProtectedRedis)
	cfg := testconfig.ProtectedRedisConfig(t)
	strg, err := mem.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	ms := &mockRedisStasher{strg: strg}
	wrapper, err := WithRedisLock(cfg.Endpoint, cfg.Password, storage.WithChecker(strg))
	if err != nil {
		t.Fatal(err)
	}
	s := wrapper(ms)

	var eg errgroup.Group
	for i := 0; i < 5; i++ {
		eg.Go(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			_, err := s.Stash(ctx, "mod", "ver")
			return err
		})
	}

	err = eg.Wait()
	if err != nil {
		t.Fatal(err)
	}
}

// Verify the WithRedisLock fails with the correct error when trying
// to connect with the wrong password.
func TestWithRedisLockWithWrongPassword(t *testing.T) {
	t.Parallel()
	testutil.CheckTestDependencies(t, testutil.TestDependencyProtectedRedis)
	cfg := testconfig.ProtectedRedisConfig(t)
	strg, err := mem.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	_, err = WithRedisLock(cfg.Endpoint, "", storage.WithChecker(strg))
	if err == nil {
		t.Fatal("Expected Connection Error")
	}

	if err.Error() != "NOAUTH Authentication required." {
		t.Fatalf("Wrong error was thrown %s\n", err.Error())
	}
}

// mockRedisStasher is like mockStasher
// but leverages in memory storage
// so that redis can determine
// whether to call the underlying stasher or not.
type mockRedisStasher struct {
	strg storage.Backend
	mu   sync.Mutex
	num  int
}

func (ms *mockRedisStasher) Stash(ctx context.Context, mod, ver string) (string, error) {
	time.Sleep(time.Millisecond * 100) // allow for second requests to come in.
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.num == 0 {
		err := ms.strg.Save(
			ctx,
			mod,
			ver,
			[]byte("mod file"),
			strings.NewReader("zip file"),
			[]byte("info file"),
		)
		if err != nil {
			return "", err
		}
		ms.num++
		return "", nil
	}
	return "", fmt.Errorf("second time error")
}
