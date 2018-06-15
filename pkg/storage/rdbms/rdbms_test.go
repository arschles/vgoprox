package rdbms

import (
	"context"
	"io/ioutil"

	"github.com/bketelsen/buffet"

	"github.com/gobuffalo/buffalo"
)

func (rd *RDBMSTestSuite) TestGetSaveListRoundTrip() {
	c := &buffalo.DefaultContext{
		Context: context.Background(),
	}
	sp := buffet.SpanFromContext(c)
	sp.SetOperationName("test.storage.rdbms.GetSaveListRoundTrip")
	defer sp.Finish()

	r := rd.Require()
	err := rd.storage.Save(c, module, version, mod, zip, info)
	r.NoError(err)
	listedVersions, err := rd.storage.List(module)
	r.NoError(err)
	r.Equal(1, len(listedVersions))
	retVersion := listedVersions[0]
	r.Equal(version, retVersion)
	gotten, err := rd.storage.Get(module, version)
	r.NoError(err)
	defer gotten.Zip.Close()
	// TODO: test the time
	r.Equal(gotten.Mod, mod)
	zipContent, err := ioutil.ReadAll(gotten.Zip)
	r.NoError(err)
	r.Equal(zipContent, zip)
	r.Equal(gotten.Info, info)
}

func (rd *RDBMSTestSuite) TestNewRDBMSStorage() {
	r := rd.Require()
	e := "development"
	getterSaver := NewRDBMSStorage(e)
	getterSaver.Connect()

	r.NotNil(getterSaver.conn)
	r.Equal(getterSaver.connectionName, e)
}
