package stash

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/technosophos/moniker"
	"golang.org/x/sync/errgroup"
)

// TestWithAzureBlob requires a real AzureBlob backend implementation
// and it will ensure that saving to modules at the same time
// is done synchronously so that only the first module gets saved.
func TestWithAzureBlob(t *testing.T) {
	containerName := randomContainerName(os.Getenv("DRONE_PULL_REQUEST"))

	cfg := getAzureTestConfig(t, containerName)
	if cfg == nil {
		t.SkipNow()
	}
	strg, err := mem.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	ms := &mockAzureBlobStasher{strg: strg}
	wpr, err := WithAzureBlobLock(cfg, time.Second*10, storage.WithChecker(strg))
	if err != nil {
		t.Fatal(err)
	}
	s := wpr(ms)

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

// mockAzureBlobStasher is like mockStasher
// but leverages in memory storage
// so that azure blob can determine
// whether to call the underlying stasher or not.
type mockAzureBlobStasher struct {
	strg storage.Backend
	mu   sync.Mutex
	num  int
}

func (ms *mockAzureBlobStasher) Stash(ctx context.Context, mod, ver string) (string, error) {
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

func getAzureTestConfig(t *testing.T, containerName string) *config.AzureBlobConfig {
	key := os.Getenv("ATHENS_AZURE_ACCOUNT_KEY")
	if key == "" {
		t.Log("ATHENS_AZURE_ACCOUNT_KEY not provided.")
		return nil
	}
	name := "athens" //os.Getenv("AZURE_ACCOUNT_NAME")
	if name == "" {
		t.Log("AZURE_ACCOUNT_NAME not provided.")
		return nil
	}
	return &config.AzureBlobConfig{
		AccountName:   name,
		AccountKey:    key,
		ContainerName: containerName,
	}
}

func randomContainerName(prefix string) string {
	// moniker is a cool library to produce mostly unique, human-readable names
	// see https://github.com/technosophos/moniker for more details
	namer := moniker.New()
	if prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, namer.NameSep(""))
	}
	return namer.NameSep("")
}
