package module

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"github.com/gomods/athens/pkg/storage"
	"github.com/spf13/afero"
)

type goGetFetcher struct {
	fs           afero.Fs
	goBinaryName string
	envVars      []string
	gogetDir     string
}

type goModule struct {
	Path     string `json:"path"`     // module path
	Version  string `json:"version"`  // module version
	Error    string `json:"error"`    // error loading module
	Info     string `json:"info"`     // absolute path to cached .info file
	GoMod    string `json:"goMod"`    // absolute path to cached .mod file
	Zip      string `json:"zip"`      // absolute path to cached .zip file
	Dir      string `json:"dir"`      // absolute path to cached source root directory
	Sum      string `json:"sum"`      // checksum for path, version (as in go.sum)
	GoModSum string `json:"goModSum"` // checksum for go.mod (as in go.sum)
}

// NewGoGetFetcher creates fetcher which uses go get tool to fetch modules
func NewGoGetFetcher(goBinaryName, gogetDir string, envVars []string, fs afero.Fs) (Fetcher, error) {
	const op errors.Op = "module.NewGoGetFetcher"
	if err := validGoBinary(goBinaryName); err != nil {
		return nil, errors.E(op, err)
	}
	return &goGetFetcher{
		fs:           fs,
		goBinaryName: goBinaryName,
		envVars:      envVars,
		gogetDir:     gogetDir,
	}, nil
}

// Fetch downloads the sources from the go binary and returns the corresponding
// .info, .mod, and .zip files.
func (g *goGetFetcher) Fetch(ctx context.Context, mod, ver string) (*storage.Version, error) {
	const op errors.Op = "goGetFetcher.Fetch"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()

	// setup the GOPATH
	goPathRoot, err := afero.TempDir(g.fs, g.gogetDir, "athens")
	if err != nil {
		return nil, errors.E(op, err)
	}
	sourcePath := filepath.Join(goPathRoot, "src")
	modPath := filepath.Join(sourcePath, getRepoDirName(mod, ver))
	if err := g.fs.MkdirAll(modPath, os.ModeDir|os.ModePerm); err != nil {
		clearFiles(g.fs, goPathRoot)
		return nil, errors.E(op, err)
	}

	m, err := downloadModule(
		ctx,
		g.goBinaryName,
		g.envVars,
		g.fs,
		goPathRoot,
		modPath,
		mod,
		ver,
	)
	if err != nil {
		clearFiles(g.fs, goPathRoot)
		return nil, errors.E(op, err)
	}

	var storageVer storage.Version
	storageVer.Semver = m.Version
	info, err := afero.ReadFile(g.fs, m.Info)
	if err != nil {
		return nil, errors.E(op, err)
	}
	storageVer.Info = info

	gomod, err := afero.ReadFile(g.fs, m.GoMod)
	if err != nil {
		return nil, errors.E(op, err)
	}
	storageVer.Mod = gomod

	zip, err := g.fs.Open(m.Zip)
	if err != nil {
		return nil, errors.E(op, err)
	}
	// note: don't close zip here so that the caller can read directly from disk.
	//
	// if we close, then the caller will panic, and the alternative to make this work is
	// that we read into memory and return an io.ReadCloser that reads out of memory
	storageVer.Zip = &zipReadCloser{zip, g.fs, goPathRoot}

	return &storageVer, nil
}

// given a filesystem, gopath, repository root, module and version, runs 'go mod download -json'
// on module@version from the repoRoot with GOPATH=gopath, and returns a non-nil error if anything went wrong.
func downloadModule(
	ctx context.Context,
	goBinaryName string,
	envVars []string,
	fs afero.Fs,
	gopath,
	repoRoot,
	module,
	version string,
) (goModule, error) {
	const op errors.Op = "module.downloadModule"

	uri := strings.TrimSuffix(module, "/")
	fullURI := fmt.Sprintf("%s@%s", uri, version)

	cmd := exec.CommandContext(ctx, goBinaryName, "mod", "download", "-json", fullURI)
	cmd.Env = prepareEnv(gopath, envVars)
	cmd.Dir = repoRoot
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("%v: %s", err, stderr)
		if gitAuthFailed(err) {
			err := fmt.Errorf("git authorization failed: %v", err)
			return goModule{}, errors.E(op, err, errors.KindUnauthorized)
		}

		var m goModule
		if jsonErr := json.NewDecoder(stdout).Decode(&m); jsonErr != nil {
			return goModule{}, errors.E(op, err)
		}
		// github quota exceeded
		if isLimitHit(m.Error) {
			return goModule{}, errors.E(op, m.Error, errors.KindRateLimit)
		}
		return goModule{}, errors.E(op, m.Error, errors.KindNotFound)
	}

	var m goModule
	if err = json.NewDecoder(stdout).Decode(&m); err != nil {
		return goModule{}, errors.E(op, err)
	}
	if m.Error != "" {
		return goModule{}, errors.E(op, m.Error)
	}

	return m, nil
}

func isLimitHit(o string) bool {
	return strings.Contains(o, "403 response from api.github.com")
}

// getRepoDirName takes a raw repository URI and a version and creates a directory name that the
// repository contents can be put into
func getRepoDirName(repoURI, version string) string {
	escapedURI := strings.Replace(repoURI, "/", "-", -1)
	return fmt.Sprintf("%s-%s", escapedURI, version)
}

func validGoBinary(name string) error {
	const op errors.Op = "module.validGoBinary"
	err := exec.Command(name).Run()
	_, ok := err.(*exec.ExitError)
	if err != nil && !ok {
		return errors.E(op, err)
	}
	return nil
}
