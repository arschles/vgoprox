package module

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"github.com/spf13/afero"
)

type goListResult struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Time    string `json:"String"`
}

// PseudoVersionFromHash returns the go mod pseudoversion associated to the given commit hash used as version
func PseudoVersionFromHash(ctx context.Context, gobinary, mod, ver string) (string, error) {
	const op errors.Op = "goGetFetcher.PseudoVersionFromHash"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()

	if IsModVersion(ver) {
		return ver, nil
	}

	uri := strings.TrimSuffix(mod, "/")
	fullURI := fmt.Sprintf("%s@%s", uri, ver)

	cmd := exec.Command(gobinary, "list", "-m", "-json", fullURI)

	fs := afero.NewOsFs()
	tmpRoot, err := afero.TempDir(fs, "", "golist")
	sourcePath := filepath.Join(tmpRoot, "src")
	modPath := filepath.Join(sourcePath, getRepoDirName(mod, ver))
	fs.MkdirAll(modPath, os.ModeDir|os.ModePerm)
	Dummy(fs, modPath)

	cmd.Env = PrepareEnv(tmpRoot)
	cmd.Dir = modPath

	o, err := cmd.Output()
	fmt.Printf(string(o))

	if err != nil {
		return "", errors.E(op, err)
	}

	var r goListResult
	err = json.Unmarshal(o, &r)
	if err != nil {
		return "", errors.E(op, err)
	}
	return r.Version, nil
}

// IsModVersion tells whether the passed string respects the semantic version pattern
func IsModVersion(ver string) bool {
	res, _ := regexp.Match("v\\d+\\.\\d+.\\d+", []byte(ver))
	return res
}
