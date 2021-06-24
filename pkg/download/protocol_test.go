package download

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/index/nop"
	"github.com/gomods/athens/pkg/module"
	"github.com/gomods/athens/pkg/stash"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

var (
	testConfigPath = filepath.Join("..", "..", "config.dev.toml")
)

func getDP(t *testing.T) Protocol {
	t.Helper()
	conf, err := config.GetConf(testConfigPath)
	if err != nil {
		t.Fatalf("Unable to parse config file: %s", err.Error())
	}
	goBin := conf.GoBinary
	fs := afero.NewOsFs()
	mf, err := module.NewGoGetFetcher(goBin, conf.GoGetDir, conf.GoBinaryEnvVars, fs)
	if err != nil {
		t.Fatal(err)
	}
	s, err := mem.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	st := stash.New(mf, s, nop.New())
	return New(&Opts{
		Storage:     s,
		Stasher:     st,
		Lister:      module.NewVCSLister(goBin, conf.GoBinaryEnvVars, fs),
		NetworkMode: Strict,
	})
}

type listTest struct {
	name string
	path string
	tags []string
}

var listTests = []listTest{
	{
		name: "happy tags",
		path: "github.com/athens-artifacts/happy-path",
		tags: []string{"v0.0.1", "v0.0.2", "v0.0.3"},
	},
	{
		name: "no tags",
		path: "github.com/athens-artifacts/no-tags",
		tags: []string{},
	},
}

func TestList(t *testing.T) {
	dp := getDP(t)
	ctx := context.Background()

	for _, tc := range listTests {
		t.Run(tc.name, func(t *testing.T) {
			versions, err := dp.List(ctx, tc.path)
			require.NoError(t, err)
			require.EqualValues(t, tc.tags, versions)
		})
	}
}

type listModeTest struct {
	name         string
	path         string
	storageTags  []string
	upstreamList []string
	upstreamErr  error
	networkmode  string
	wantTags     []string
	wantErr      bool
}

var listModeTests = []listModeTest{
	{
		name:        "strict no tags",
		networkmode: Offline,
		path:        "github.com/athens-artifacts/happy-path",
		wantTags:    []string{},
	},
	{
		name:         "strict tags",
		networkmode:  Strict,
		path:         "github.com/athens-artifacts/happy-path",
		storageTags:  []string{"v0.0.4"},
		upstreamList: []string{"v0.0.1", "v0.0.2", "v0.0.3"},
		wantTags:     []string{"v0.0.1", "v0.0.2", "v0.0.3", "v0.0.4"},
	},
	{
		name:        "offline",
		networkmode: Offline,
		path:        "github.com/athens-artifacts/happy-path",
		storageTags: []string{"v0.0.4"},
		wantTags:    []string{"v0.0.4"},
	},
	{
		name:         "fallback with err",
		networkmode:  Fallback,
		path:         "github.com/athens-artifacts/happy-path",
		storageTags:  []string{"v0.0.4"},
		upstreamList: []string{},
		upstreamErr:  errors.E("test", "unexpected error"),
		wantTags:     []string{"v0.0.4"},
	},
	{
		name:         "fallback upstream not found",
		networkmode:  Fallback,
		path:         "github.com/athens-artifacts/happy-path",
		storageTags:  []string{"v0.0.4"},
		upstreamList: []string{},
		upstreamErr:  errors.E("test", "remote: Repository not found", errors.KindNotFound),
		wantTags:     []string{"v0.0.4"},
	},
	{
		name:         "fallback error with no storage",
		networkmode:  Fallback,
		path:         "github.com/athens-artifacts/happy-path",
		storageTags:  []string{},
		upstreamList: []string{},
		upstreamErr:  errors.E("test", "remote: Repository not found", errors.KindNotFound),
		wantTags:     nil,
		wantErr:      true,
	},
}

func TestListMode(t *testing.T) {
	ctx := context.Background()
	for _, tc := range listModeTests {
		strg, err := mem.NewStorage()
		require.NoError(t, err)
		ml := &mockLister{
			list: tc.upstreamList,
			err:  tc.upstreamErr,
		}
		dp := &protocol{
			storage:     strg,
			lister:      ml,
			networkMode: tc.networkmode,
		}
		for _, tag := range tc.storageTags {
			err := strg.Save(ctx, tc.path, tag, []byte("mod"), bytes.NewReader([]byte("zip")), []byte("info"))
			require.NoError(t, err)
		}
		t.Run(tc.name, func(t *testing.T) {
			versions, err := dp.List(ctx, tc.path)
			if err != nil && !tc.wantErr {
				t.Fatal(err)
			}
			require.EqualValues(t, tc.wantTags, versions)
			if tc.networkmode == Offline && ml.called {
				t.Fatal("upstream lister must not be called in offline mode")
			}
		})
	}
}

func TestConcurrentLists(t *testing.T) {
	dp := getDP(t)
	ctx := context.Background()

	pkg := "github.com/athens-artifacts/samplelib"
	var pkgErr error

	subPkg := "github.com/athens-artifacts/samplelib/types"
	var subPkgErr error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		_, pkgErr = dp.List(ctx, pkg)
		wg.Done()
	}()
	go func() {
		_, subPkgErr = dp.List(ctx, subPkg)
		wg.Done()
	}()
	wg.Wait()

	if pkgErr != nil {
		t.Fatalf("expected version listing of %v to succeed but got %v", pkg, pkgErr)
	}

	if subPkgErr == nil {
		t.Fatalf("expected version listing of %v to fail because it's a subdirectory", subPkg)
	}
}

type latestTest struct {
	name string
	path string
	info *storage.RevInfo
	err  bool
}

var latestTests = []latestTest{
	{
		name: "happy path",
		path: "github.com/athens-artifacts/no-tags",
		info: &storage.RevInfo{
			Version: "v0.0.0-20180803171426-1a540c5d67ab",
			Time:    time.Date(2018, 8, 3, 17, 14, 26, 0, time.UTC),
		},
	},
	{
		name: "tagged latest",
		path: "github.com/athens-artifacts/happy-path",
		info: &storage.RevInfo{
			Version: "v0.0.3",
			Time:    time.Date(2018, 8, 3, 17, 16, 00, 0, time.UTC),
		},
	},
}

func TestLatest(t *testing.T) {
	dp := getDP(t)
	ctx := context.Background()

	for _, tc := range latestTests {
		t.Run(tc.name, func(t *testing.T) {
			info, err := dp.Latest(ctx, tc.path)
			if !tc.err && err != nil {
				t.Fatal(err)
			} else if tc.err && err == nil {
				t.Fatalf("expected %v error but got nil", tc.err)
			}

			require.EqualValues(t, tc.info, info)
		})
	}
}

type infoTest struct {
	name    string
	path    string
	version string
	info    *storage.RevInfo
}

var infoTests = []infoTest{
	{
		name:    "happy path",
		path:    "github.com/athens-artifacts/happy-path",
		version: "v0.0.2",
		info: &storage.RevInfo{
			Version: "v0.0.2",
			Time:    time.Date(2018, 8, 3, 3, 45, 19, 0, time.UTC),
		},
	},
	{
		name:    "pseudo version",
		path:    "github.com/athens-artifacts/no-tags",
		version: "v0.0.0-20180803035119-e4e0177efdb5",
		info: &storage.RevInfo{
			Version: "v0.0.0-20180803035119-e4e0177efdb5",
			Time:    time.Date(2018, 8, 3, 3, 51, 19, 0, time.UTC),
		},
	},
}

func TestInfo(t *testing.T) {
	dp := getDP(t)
	ctx := context.Background()

	for _, tc := range infoTests {
		t.Run(tc.name, func(t *testing.T) {
			bts, err := dp.Info(ctx, tc.path, tc.version)
			require.NoError(t, err)

			var info storage.RevInfo
			dec := json.NewDecoder(bytes.NewReader(bts))
			dec.DisallowUnknownFields()
			err = dec.Decode(&info)
			require.NoError(t, err)

			require.EqualValues(t, tc.info, &info)
		})
	}
}

type modTest struct {
	name    string
	path    string
	version string
	err     bool
}

var modTests = []modTest{
	{
		name:    "no mod file",
		path:    "github.com/athens-artifacts/no-tags",
		version: "v0.0.0-20180803035119-e4e0177efdb5",
	},
	{
		name:    "upstream mod file",
		path:    "github.com/athens-artifacts/happy-path",
		version: "v0.0.3",
	},
	{
		name:    "incorrect github repo",
		path:    "github.com/athens-artifacts/not-exists",
		version: "v1.0.0",
		err:     true,
	},
}

func rmNewLine(input string) string {
	re := regexp.MustCompile(`\r?\n`)
	return re.ReplaceAllString(input, "")
}

func TestGoMod(t *testing.T) {
	dp := getDP(t)
	ctx := context.Background()

	for _, tc := range modTests {
		t.Run(tc.name, func(t *testing.T) {
			mod, err := dp.GoMod(ctx, tc.path, tc.version)
			require.Equal(t, tc.err, err != nil, err)

			if tc.err {
				t.Skip()
			}
			expected := rmNewLine(string(getGoldenFile(t, tc.name)))
			res := rmNewLine(string(mod))
			require.Equal(t, expected, res)
		})
	}
}

func getGoldenFile(t *testing.T, name string) []byte {
	t.Helper()
	file := filepath.Join("test_data", strings.Replace(name, " ", "_", -1)+".golden")
	bts, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	return bts
}

type testMod struct {
	mod, ver string
}

var mods = []testMod{
	{"github.com/athens-artifacts/no-tags", "v0.0.2"},
	{"github.com/athens-artifacts/happy-path", "v0.0.0-20180803035119-e4e0177efdb5"},
	{"github.com/athens-artifacts/samplelib", "v1.0.0"},
}

func TestDownloadProtocol(t *testing.T) {
	s, err := mem.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	mp := &mockFetcher{}
	st := stash.New(mp, s, nop.New())
	dp := New(&Opts{s, st, nil, nil, Strict})
	ctx := context.Background()

	var eg errgroup.Group
	for i := 0; i < len(mods); i++ {
		m := mods[i]
		eg.Go(func() error {
			_, err := dp.GoMod(ctx, m.mod, m.ver)
			return err
		})
	}

	err = eg.Wait()
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range mods {
		bts, err := dp.GoMod(ctx, m.mod, m.ver)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(bts, []byte(m.mod+"@"+m.ver)) {
			t.Fatalf("unexpected gomod content: %s", bts)
		}
	}
}

type mockFetcher struct{}

func (m *mockFetcher) Fetch(ctx context.Context, mod, ver string) (*storage.Version, error) {
	bts := []byte(mod + "@" + ver)
	return &storage.Version{
		Mod:  bts,
		Info: bts,
		Zip:  ioutil.NopCloser(bytes.NewReader(bts)),
	}, nil
}

func TestDownloadProtocolWhenFetchFails(t *testing.T) {
	s, err := mem.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	fakeMod := testMod{"github.com/athens-artifacts/samplelib", "v1.0.0"}
	bts := []byte(fakeMod.mod + "@" + fakeMod.ver)
	err = s.Save(context.Background(), fakeMod.mod, fakeMod.ver, bts, ioutil.NopCloser(bytes.NewReader(bts)), bts)
	if err != nil {
		t.Fatal(err)
	}
	mp := &notFoundFetcher{}
	st := stash.New(mp, s, nop.New())
	dp := New(&Opts{s, st, nil, nil, Strict})
	ctx := context.Background()
	_, err = dp.GoMod(ctx, fakeMod.mod, fakeMod.ver)
	if err != nil {
		t.Errorf("Download protocol should succeed, instead it gave error %s \n", err)
	}
}

func TestAsyncRedirect(t *testing.T) {
	s, err := mem.NewStorage()
	require.NoError(t, err)
	ms := &mockStasher{s, make(chan bool)}
	dp := New(&Opts{
		Stasher: ms,
		Storage: s,
		DownloadFile: &mode.DownloadFile{
			Mode:        mode.Async,
			DownloadURL: "https://gomods.io",
		},
	})
	mod, ver := "github.com/athens-artifacts/happy-path", "v0.0.1"
	_, err = dp.Info(context.Background(), mod, ver)
	if errors.Kind(err) != errors.KindNotFound {
		t.Fatalf("expected async_redirect to enforce a 404 but got %v", errors.Kind(err))
	}
	<-ms.ch
	info, err := dp.Info(context.Background(), mod, ver)
	require.NoError(t, err)
	require.Equal(t, string(info), "info", "expected async fetch to be successful")
}

type mockStasher struct {
	s  storage.Backend
	ch chan bool
}

func (ms *mockStasher) Stash(ctx context.Context, mod string, ver string) (string, error) {
	err := ms.s.Save(ctx, mod, ver, []byte("mod"), strings.NewReader("zip"), []byte("info"))
	ms.ch <- true // signal async stashing is done
	return ver, err
}

type notFoundFetcher struct{}

func (m *notFoundFetcher) Fetch(ctx context.Context, mod, ver string) (*storage.Version, error) {
	const op errors.Op = "goGetFetcher.Fetch"
	return nil, errors.E(op, "Fetcher error")
}

type mockLister struct {
	called bool
	list   []string
	err    error
}

func (ml *mockLister) List(ctx context.Context, mod string) (*storage.RevInfo, []string, error) {
	ml.called = true
	return nil, ml.list, ml.err
}
