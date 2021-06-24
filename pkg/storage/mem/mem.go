package mem

import (
	"fmt"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/fs"
	"github.com/spf13/afero"
)

// NewStorage creates new in-memory storage using the afero.NewMemMapFs() in memory file system
func NewStorage() (storage.Backend, error) {
	const op errors.Op = "mem.NewStorage"

	memFs := afero.NewMemMapFs()
	tmpDir, err := afero.TempDir(memFs, "", "")
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("could not create temp dir for 'In Memory' storage (%s)", err))
	}

	memStorage, err := fs.NewStorage(tmpDir, memFs)
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("could not create storage from memory fs (%s)", err))
	}
	return memStorage, nil
}
