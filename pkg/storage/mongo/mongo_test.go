package mongo

import (
	"context"
	"os"
	"testing"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/storage/compliance"
	"github.com/stretchr/testify/require"
)

func TestBackend(t *testing.T) {
	backend := getStorage(t)
	compliance.RunTests(t, backend, backend.clear)
}

func (m *ModuleStore) clear() error {
	m.client.Database(m.db).Drop(context.Background())
	return nil
}

func BenchmarkBackend(b *testing.B) {
	backend := getStorage(b)
	compliance.RunBenchmarks(b, backend, backend.clear)
}

func getStorage(tb testing.TB) *ModuleStore {
	url := os.Getenv("ATHENS_MONGO_STORAGE_URL")

	if url == "" {
		tb.SkipNow()
	}

	backend, err := NewStorage(&config.MongoConfig{URL: url}, config.GetTimeoutDuration(300))
	require.NoError(tb, err)

	return backend
}

func TestNewStorageWithDefaultOverrides(t *testing.T) {
	url := os.Getenv("ATHENS_MONGO_STORAGE_URL")

	if url == "" {
		t.SkipNow()
	}

	testCases := []struct {
		name        string
		dbName      string
		expDbName   string
		collName    string
		expCollName string
	}{
		{"Test Default 'Athens' DB Name", "athens", "athens", "modules", "modules"},          //Tests the default database name
		{"Test Custom DB Name", "testAthens", "testAthens", "modules", "modules"},            //Tests a non-default database name
		{"Test Blank DB Name", "", "athens", "modules", "modules"},                           //Tests the blank database name edge-case
		{"Test Default 'Modules' Collection Name", "athens", "athens", "modules", "modules"}, //Tests the default collection name
		{"Test Custom Collection Name", "athens", "athens", "testModules", "testModules"},    //Tests the non-default collection name
		{"Test Blank Collection Name", "athens", "athens", "", "modules"},                    //Tests the blank collection name edge-case

	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			backend, err := NewStorage(&config.MongoConfig{URL: url, DefaultDBName: test.dbName, DefaultCollectionName: test.collName}, config.GetTimeoutDuration(300))
			require.NoError(t, err)
			require.Equal(t, test.expDbName, backend.db)
			require.Equal(t, test.expCollName, backend.coll)

		})
	}
}
