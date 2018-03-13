package memory

import (
	"bytes"
	"io/ioutil"
	"time"

	"github.com/gomods/athens/pkg/storage"
)

func (v *getterSaverImpl) Save(baseURL, module, vsn string, mod, zip []byte) error {
	reader := ioutil.NopCloser(bytes.NewReader(zip))
	newVsn := &storage.Version{
		RevInfo: storage.RevInfo{
			Version: vsn,
			Short:   vsn,
			Time:    time.Now(),
			Name:    vsn,
		},
		Mod: mod,
		Zip: reader,
	}
	v.Lock()
	defer v.Unlock()
	key := v.key(baseURL, module)
	existingVersionsSlice := v.versions[key]
	for _, version := range existingVersionsSlice {
		if version.RevInfo.Version == vsn {
			return storage.ErrVersionAlreadyExists{
				BasePath: baseURL,
				Module:   module,
				Version:  vsn,
			}
		}
	}
	newVersionsSlice := append(existingVersionsSlice, newVsn)
	v.versions[key] = newVersionsSlice
	return nil
}
