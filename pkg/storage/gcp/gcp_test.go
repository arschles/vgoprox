package gcp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/storage/compliance"
	"github.com/stretchr/testify/require"
	"github.com/technosophos/moniker"
	"google.golang.org/api/iterator"
)

func TestBackend(t *testing.T) {
	backend := getStorage(t)
	defer backend.bucket.Delete(context.Background())
	compliance.RunTests(t, backend, backend.clear)
}

func BenchmarkBackend(b *testing.B) {
	backend := getStorage(b)
	defer backend.bucket.Delete(context.Background())
	compliance.RunBenchmarks(b, backend, backend.clear)
}

func (s *Storage) clear() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	it := s.bucket.Objects(ctx, nil)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		err = s.bucket.Object(attrs.Name).Delete(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func getStorage(t testing.TB) *Storage {
	t.Helper()

	server := fakestorage.NewServer(nil)

	cfg := &config.GCPConfig{
		Bucket: randomBucketName("bucket_time"),
	}
	client, err := newClient(context.Background(), cfg, 30*time.Second, server.HTTPClient())
	require.NoError(t, err)
	err = client.bucket.Create(context.Background(), "project_time", nil)
	require.NoError(t, err)
	return client
}

func randomBucketName(prefix string) string {
	// moniker is a cool library to produce mostly unique, human-readable names
	// see https://github.com/technosophos/moniker for more details
	namer := moniker.New()
	if prefix != "" {
		return fmt.Sprintf("%s_%s", prefix, namer.NameSep("_"))
	}
	return namer.NameSep("_")
}
