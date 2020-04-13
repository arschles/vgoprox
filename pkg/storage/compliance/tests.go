package compliance

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sort"
	"testing"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/storage"
	"github.com/stretchr/testify/require"
)

// RunTests takes a backend implementation and runs compliance tests
// against the interface.
func RunTests(t *testing.T, b storage.Backend, clearBackend func() error) {
	require.NoError(t, clearBackend(), "pre-clearing backend failed")
	defer require.NoError(t, clearBackend(), "post-clearing backend failed")
	testNotFound(t, b)
	testList(t, b)
	testListSuffix(t, b)
	testDelete(t, b)
	testGet(t, b)
	testExists(t, b)
	testShouldNotExist(t, b)
	// testCatalog(t, b)
}

// testNotFound ensures that a storage Backend
// returns a KindNotFound error when asking for
// non existing modules.
func testNotFound(t *testing.T, b storage.Backend) {
	mod, ver := "github.com/gomods/athens", "yyy"
	ctx := context.Background()

	err := b.Delete(ctx, mod, ver)
	require.Error(t, err)
	require.Equal(t, errors.KindNotFound, errors.Kind(err))

	_, err = b.GoMod(ctx, mod, ver)
	require.Error(t, err)
	require.Equal(t, errors.KindNotFound, errors.Kind(err))

	_, err = b.Info(ctx, mod, ver)
	require.Error(t, err)
	require.Equal(t, errors.KindNotFound, errors.Kind(err))

	vs, err := b.List(ctx, mod)
	require.NoError(t, err)
	require.Equal(t, 0, len(vs))

	_, err = b.Zip(ctx, mod, ver)
	require.Error(t, err)
	require.Equal(t, errors.KindNotFound, errors.Kind(err))
}

// testListPrefixes makes sure that if you have two modules, such as
// github.com/one/two and github.com/one/two-suffix, then the versions
// should not be mixed just because they share a similar prefix.
func testListSuffix(t *testing.T, b storage.Backend) {
	ctx := context.Background()

	modVers := map[string][]string{
		"github.com/one/two":       {"v1.1.0", "v1.2.0", "v1.3.0"},
		"github.com/one/two/v2":    {"v2.1.0"},
		"github.com/one/two-other": {"v0.9.0"},
		"github.com/one":           {}, // not a module but a valid query, so no versions
	}
	for modname, versions := range modVers {
		for _, version := range versions {
			mock := getMockModule()
			err := b.Save(
				ctx,
				modname,
				version,
				mock.Mod,
				mock.Zip,
				mock.Info,
			)
			require.NoError(t, err, "Save for storage failed")
		}
	}
	defer func() {
		for modname, versions := range modVers {
			for _, version := range versions {
				b.Delete(ctx, modname, version)
			}
		}
	}()
	for modname, versions := range modVers {
		retVersions, err := b.List(ctx, modname)
		require.NoError(t, err)
		if len(versions) == 0 {
			require.Empty(t, retVersions)
		} else {
			require.Equal(t, versions, retVersions)
		}
	}
}

// testList tests that a storage Backend returns
// the exact list of versions that are saved.
func testList(t *testing.T, b storage.Backend) {
	ctx := context.Background()

	modname := "github.com/gomods/athens"
	versions := []string{"v1.1.0", "v1.2.0", "v1.3.0"}
	for _, version := range versions {
		mock := getMockModule()
		err := b.Save(
			ctx,
			modname,
			version,
			mock.Mod,
			mock.Zip,
			mock.Info,
		)
		require.NoError(t, err, "Save for storage failed")
	}
	defer func() {
		for _, ver := range versions {
			b.Delete(ctx, modname, ver)
		}
	}()
	retVersions, err := b.List(ctx, modname)
	require.NoError(t, err)
	require.Equal(t, versions, retVersions)
}

// testGet saves and retrieves a module successfully.
func testGet(t *testing.T, b storage.Backend) {
	ctx := context.Background()
	modname := "github.com/gomods/athens"
	ver := "v1.2.3"
	mock := getMockModule()
	zipBts, _ := ioutil.ReadAll(mock.Zip)
	b.Save(ctx, modname, ver, mock.Mod, bytes.NewReader(zipBts), mock.Info)
	defer b.Delete(ctx, modname, ver)

	info, err := b.Info(ctx, modname, ver)
	require.NoError(t, err)
	require.Equal(t, mock.Info, info)

	mod, err := b.GoMod(ctx, modname, ver)
	require.NoError(t, err)
	require.Equal(t, string(mock.Mod), string(mod))

	zip, err := b.Zip(ctx, modname, ver)
	require.NoError(t, err)
	givenZipBts, err := ioutil.ReadAll(zip)
	require.NoError(t, err)
	require.Equal(t, zipBts, givenZipBts)
}

func testExists(t *testing.T, b storage.Backend) {
	ctx := context.Background()
	modname := "github.com/gomods/athens"
	ver := "v1.2.3"
	mock := getMockModule()
	zipBts, _ := ioutil.ReadAll(mock.Zip)
	b.Save(ctx, modname, ver, mock.Mod, bytes.NewReader(zipBts), mock.Info)
	defer b.Delete(ctx, modname, ver)
	checker := storage.WithChecker(b)
	exists, err := checker.Exists(ctx, modname, ver)
	require.NoError(t, err)
	require.Equal(t, true, exists)
}

func testShouldNotExist(t *testing.T, b storage.Backend) {
	ctx := context.Background()
	mod := "github.com/gomods/shouldNotExist"
	ver := "v1.2.3-pre.1"
	mock := getMockModule()
	zipBts, _ := ioutil.ReadAll(mock.Zip)
	err := b.Save(ctx, mod, ver, mock.Mod, bytes.NewReader(zipBts), mock.Info)
	require.NoError(t, err, "should successfully safe a mock module")
	defer b.Delete(ctx, mod, ver)

	prefixVer := "v1.2.3-pre"

	exists, err := storage.WithChecker(b).Exists(ctx, mod, prefixVer)
	require.NoError(t, err)
	if exists {
		t.Fatal("a non existing version that has the same prefix of an existing version should not exist")
	}
}

// testDelete tests that a module can be deleted from a
// storage Backend and the Exists method returns false
// afterwards.
func testDelete(t *testing.T, b storage.Backend) {
	ctx := context.Background()
	modname := "github.com/gomods/athens"
	version := fmt.Sprintf("%s%d", "delete", rand.Int())

	mock := getMockModule()
	err := b.Save(ctx, modname, version, mock.Mod, mock.Zip, mock.Info)
	require.NoError(t, err)

	err = b.Delete(ctx, modname, version)
	require.NoError(t, err)

	exists, err := storage.WithChecker(b).Exists(ctx, modname, version)
	require.NoError(t, err)
	require.Equal(t, false, exists)
}

func testCatalog(t *testing.T, b storage.Backend) {
	cs, ok := b.(storage.Cataloger)
	if !ok {
		t.Skip()
	}
	ctx := context.Background()

	mock := getMockModule()
	zipBts, _ := ioutil.ReadAll(mock.Zip)
	modname := "github.com/gomods/testCatalogModule"
	for i := 0; i < 6; i++ {
		ver := fmt.Sprintf("v1.2.%04d", i)
		b.Save(ctx, modname, ver, mock.Mod, bytes.NewReader(zipBts), mock.Info)

		defer b.Delete(ctx, modname, ver)
	}

	allres, next, err := cs.Catalog(ctx, "", 5)

	require.NoError(t, err)
	require.Equal(t, 5, len(allres))

	res, next, err := cs.Catalog(ctx, next, 50)
	allres = append(allres, res...)
	require.NoError(t, err)
	require.Equal(t, 1, len(res))
	require.Equal(t, "", next)

	sort.Slice(allres, func(i, j int) bool {
		if allres[i].Module == allres[j].Module {
			return allres[i].Version < allres[j].Version
		}
		return allres[i].Module < allres[j].Module
	})
	require.Equal(t, modname, allres[0].Module)
	require.Equal(t, "v1.2.0000", allres[0].Version)
	require.Equal(t, "v1.2.0004", allres[4].Version)

	for i := 1; i < len(allres); i++ {
		require.NotEqual(t, allres[i].Version, allres[i-1].Version)
	}
}

func getMockModule() *storage.Version {
	return &storage.Version{
		Info: []byte("123"),
		Mod:  []byte("456"),
		Zip:  ioutil.NopCloser(bytes.NewReader([]byte("789"))),
	}
}
