package mem

import (
	"fmt"
	"sync"

	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/fs"
	"github.com/spf13/afero"
)

var (
	l          sync.Mutex
	memStorage storage.BackendConnector
)

// NewStorage creates new in-memory storage using the afero.NewMemMapFs() in memory file system
func NewStorage() (storage.BackendConnector, error) {
	l.Lock()
	defer l.Unlock()

	if memStorage != nil {
		return memStorage, nil
	}

	memFs := afero.NewMemMapFs()
	tmpDir, err := afero.TempDir(memFs, "", "")
	if err != nil {
		return nil, fmt.Errorf("could not create temp dir for 'In Memory' storage (%s)", err)
	}

	s := fs.NewStorage(tmpDir, memFs)
	memStorage = storage.NoOpBackendConnector(s)
	return memStorage, nil
}
