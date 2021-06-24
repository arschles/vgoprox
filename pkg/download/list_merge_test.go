package download

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"

	athenserr "github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/stretchr/testify/require"
)

const testOp athenserr.Op = "vcsLister.List"
const testModName = "happy tags"

type listMergeTest struct {
	name        string
	newStorage  func() (storage.Backend, error)
	module      string
	goVersions  []string
	goErr       error
	strVersions []string
	strErr      error
	expected    []string
	expectedErr error
}

type storageMock struct {
	storage.Backend
	versions []string
	err      error
}

func (s *storageMock) List(ctx context.Context, module string) ([]string, error) {
	return s.versions, s.err
}

var listMergeTests = []listMergeTest{
	{
		name:        "go list full and storage full",
		newStorage:  mem.NewStorage,
		goVersions:  []string{"v1.0.0", "v1.0.2", "v1.0.3"},
		goErr:       nil,
		strVersions: []string{"v1.0.0", "v1.0.1", "v1.0.2"},
		expected:    []string{"v1.0.0", "v1.0.1", "v1.0.2", "v1.0.3"},
		expectedErr: nil,
	},
	{
		name:        "go list full and storage empty",
		newStorage:  mem.NewStorage,
		goVersions:  []string{"v1.0.0", "v1.0.1", "v1.0.2"},
		goErr:       nil,
		strVersions: []string{},
		strErr:      nil,
		expected:    []string{"v1.0.0", "v1.0.1", "v1.0.2"},
		expectedErr: nil,
	},
	{
		name:        "go list repo not found and storage full",
		newStorage:  mem.NewStorage,
		goVersions:  nil,
		goErr:       errors.New("remote: Repository not found"),
		strVersions: []string{"v1.0.0", "v1.0.1", "v1.0.2"},
		strErr:      nil,
		expected:    []string{"v1.0.0", "v1.0.1", "v1.0.2"},
		expectedErr: nil,
	},
	{
		name:        "go list repo not found and storage empty",
		newStorage:  mem.NewStorage,
		goVersions:  nil,
		goErr:       errors.New("remote: Repository not found"),
		strVersions: []string{},
		strErr:      nil,
		expected:    nil,
		expectedErr: athenserr.E(testOp, athenserr.M(testModName), athenserr.KindNotFound, errors.New("remote: Repository not found")),
	},
	{
		name:        "unexpected go err",
		newStorage:  mem.NewStorage,
		goVersions:  nil,
		goErr:       errors.New("unexpected error"),
		strVersions: []string{"1.1.1"},
		strErr:      nil,
		expected:    nil,
		expectedErr: athenserr.E(testOp, errors.New("unexpected error")),
	},
	{
		name:        "unexpected storage err",
		newStorage:  func() (storage.Backend, error) { return &storageMock{err: errors.New("unexpected error")}, nil },
		goVersions:  []string{"1.1.1"},
		goErr:       nil,
		strVersions: nil,
		strErr:      errors.New("unexpected error"),
		expected:    nil,
		expectedErr: athenserr.E(testOp, errors.New("unexpected error")),
	},
}

type listerMock struct {
	versions []string
	err      error
}

func (l *listerMock) List(ctx context.Context, mod string) (*storage.RevInfo, []string, error) {
	return nil, l.versions, l.err
}

func TestListMerge(t *testing.T) {
	ctx := context.Background()
	bts := []byte("123")
	clearStorage := func(st storage.Backend, module string, versions []string) {
		for _, v := range versions {
			st.Delete(ctx, module, v)
		}
	}

	for _, tc := range listMergeTests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.newStorage()
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range tc.strVersions {
				s.Save(ctx, testModName, v, bts, ioutil.NopCloser(bytes.NewReader(bts)), bts)
			}
			defer clearStorage(s, testModName, tc.strVersions)
			dp := New(&Opts{s, nil, &listerMock{versions: tc.goVersions, err: tc.goErr}, nil, Strict})
			list, err := dp.List(ctx, testModName)

			if ok := testErrEq(tc.expectedErr, err); !ok {
				t.Fatalf("expected err: %v, got: %v", tc.expectedErr, err)
			}
			if tc.expectedErr != nil {
				require.Equal(t, athenserr.Kind(tc.expectedErr), athenserr.Kind(err))
			}
			require.ElementsMatch(t, tc.expected, list, "expected list: %v, got: %v", tc.expected, list)
		})
	}
}

func testErrEq(a, b error) bool {
	if a == nil && b == nil {
		return true
	}

	if (a == nil) != (b == nil) {
		return false
	}

	if a.Error() != b.Error() {
		return false
	}
	return true
}
