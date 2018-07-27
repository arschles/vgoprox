package azurecdn

import (
	"github.com/gobuffalo/suite"
	"github.com/gomods/athens/pkg/storage"
	"github.com/spf13/afero"
)

// TestSuite implements storage.TestSuite interface
type TestSuite struct {
	*suite.Model
	storage storage.Backend
	fs      afero.Fs
	rootDir string
}

// NewTestSuite creates a common test suite
func NewTestSuite(model *suite.Model) (storage.TestSuite, error) {

	return &TestSuite{
		Model:   model,
		fs:      memFs,
		rootDir: r,
		storage: fsStore,
	}, nil
}

// Storage retrieves initialized storage backend
func (ts *TestSuite) Storage() storage.Backend {
	return ts.storage
}

// StorageHumanReadableName retrieves readable identifier of the storage
func (ts *TestSuite) StorageHumanReadableName() string {
	return "FileSystem"
}

// Cleanup tears down test
func (ts *TestSuite) Cleanup() {
	ts.Require().NoError(ts.fs.RemoveAll(ts.rootDir))
}
