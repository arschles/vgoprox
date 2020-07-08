package azureblob

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gomods/athens/internal/testutil"
	"github.com/gomods/athens/internal/testutil/testconfig"
	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/storage/compliance"
	"github.com/technosophos/moniker"
)

func TestBackend(t *testing.T) {
	backend := getStorage(t)
	defer backend.client.containerURL.Delete(context.Background(), azblob.ContainerAccessConditions{})
	compliance.RunTests(t, backend, backend.clear)
}

func BenchmarkBackend(b *testing.B) {
	backend := getStorage(b)
	defer backend.client.containerURL.Delete(context.Background(), azblob.ContainerAccessConditions{})
	compliance.RunBenchmarks(b, backend, backend.clear)
}

func (s *Storage) clear() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := s.client.containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		if err != nil {
			return err
		}
		marker = listBlob.NextMarker

		for _, blob := range listBlob.Segment.BlobItems {

			blobURL := s.client.containerURL.NewBlockBlobURL(blob.Name)
			_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getStorage(t testing.TB) *Storage {
	t.Helper()
	testutil.CheckTestDependencies(t, testutil.TestDependencyAzurite)
	containerName := randomContainerName(os.Getenv("DRONE_PULL_REQUEST"))
	cfg := testconfig.LoadTestConfig(t).Storage.AzureBlob
	cfg.ContainerName = containerName

	s, err := New(cfg, config.GetTimeoutDuration(30))
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.client.containerURL.Create(context.Background(), azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func randomContainerName(prefix string) string {
	// moniker is a cool library to produce mostly unique, human-readable names
	// see https://github.com/technosophos/moniker for more details
	namer := moniker.New()
	if prefix != "" {
		return fmt.Sprintf("%s_%s", prefix, namer.NameSep(""))
	}
	return namer.NameSep("")
}
