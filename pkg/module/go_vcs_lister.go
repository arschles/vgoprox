package module

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"github.com/gomods/athens/pkg/storage"
	"github.com/spf13/afero"
)

type listResp struct {
	Path     string
	Version  string
	Versions []string `json:",omitempty"`
	Time     time.Time
}

type vcsLister struct {
	goBinPath string
	env       []string
	fs        afero.Fs
}

// NewVCSLister creates an UpstreamLister which uses VCS to fetch a list of available versions
func NewVCSLister(goBinPath string, env []string, fs afero.Fs) UpstreamLister {
	return &vcsLister{
		goBinPath: goBinPath,
		env:       env,
		fs:        fs,
	}
}

func (l *vcsLister) List(ctx context.Context, module string) (*storage.RevInfo, []string, error) {
	const op errors.Op = "vcsLister.List"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()
	tmpDir, err := afero.TempDir(l.fs, "", "go-list")
	if err != nil {
		return nil, nil, errors.E(op, err)
	}
	defer l.fs.RemoveAll(tmpDir)

	cmd := exec.Command(
		l.goBinPath,
		"list", "-m", "-versions", "-json",
		config.FmtModVer(module, "latest"),
	)
	cmd.Dir = tmpDir
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	gopath, err := afero.TempDir(l.fs, "", "athens")
	if err != nil {
		return nil, nil, errors.E(op, err)
	}
	defer clearFiles(l.fs, gopath)
	cmd.Env = prepareEnv(gopath, l.env)

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("%v: %s", err, stderr)
		if gitAuthFailed(err) {
			err := fmt.Errorf("git authorization failed: %v", err)
			return nil, nil, errors.E(op, err, errors.KindUnauthorized)
		}
		// as of now, we can't recognize between a true NotFound
		// and an unexpected error, so we choose the more
		// hopeful path of NotFound. This way the Go command
		// will not log en error and we still get to log
		// what happened here if someone wants to dig in more.
		// Once, https://github.com/golang/go/issues/30134 is
		// resolved, we can hopefully differentiate.
		return nil, nil, errors.E(op, err, errors.KindNotFound)
	}

	var lr listResp
	err = json.NewDecoder(stdout).Decode(&lr)
	if err != nil {
		return nil, nil, errors.E(op, err)
	}
	rev := storage.RevInfo{
		Time:    lr.Time,
		Version: lr.Version,
	}
	return &rev, lr.Versions, nil
}

func gitAuthFailed(errCmd error) bool {
	return strings.Contains(errCmd.Error(), "fatal: could not read Username for 'https://github.com'")
}
