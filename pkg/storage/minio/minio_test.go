package minio

import (
	"bytes"
	"context"
	"io/ioutil"
)

func (d *MinioTests) TestGetSaveListRoundTrip() {
	r := d.Require()
	r.NoError(d.storage.Save(context.Background(), module, version, mod, bytes.NewReader(zip), info))
	listedVersions, err := d.storage.List(module)
	r.NoError(err)
	r.Equal(1, len(listedVersions))
	retVersion := listedVersions[0]
	r.Equal(version, retVersion)
	gotten, err := d.storage.Get(module, version)
	r.NoError(err)
	defer gotten.Zip.Close()
	// TODO: test the time
	r.Equal(gotten.Mod, mod)
	zipContent, err := ioutil.ReadAll(gotten.Zip)
	r.NoError(err)
	r.Equal(zipContent, zip)
	r.Equal(gotten.Info, info)
}
