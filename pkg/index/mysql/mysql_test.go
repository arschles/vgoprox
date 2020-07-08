package mysql

import (
	"testing"

	"github.com/gomods/athens/internal/testutil"
	"github.com/gomods/athens/internal/testutil/testconfig"
	"github.com/gomods/athens/pkg/index/compliance"
)

func TestMySQL(t *testing.T) {
	testutil.CheckTestDependencies(t, testutil.TestDependencyMySQL)
	cfg := testconfig.LoadTestConfig(t).Index.MySQL
	i, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	compliance.RunTests(t, i, i.(*indexer).clear)
}

func (i *indexer) clear() error {
	_, err := i.db.Exec(`DELETE FROM indexes`)
	return err
}
